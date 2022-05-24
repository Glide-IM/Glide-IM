package route

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/glide-im/glideim/im/api/comm"
	"github.com/glide-im/glideim/im/message"
	"reflect"
	"strings"
	"time"
)

const PathSeparator = "."

var ErrNotRouteMatches = errors.New("no route matches")

var typeRequestInfo = reflect.TypeOf((*Context)(nil))

type HandleFunc interface{}

type Validatable interface {
	Validate(data interface{}) error
}

type path struct {
	split []string
	index int
}

func newPath(action string) path {
	split := strings.Split(action, ".")
	return path{
		split: split,
		index: 0,
	}
}

func (p *path) next() (string, bool) {
	if p.index >= len(p.split) {
		return "", false
	}
	ret := p.split[p.index]
	p.index++
	return ret, true
}

type Context struct {
	Uid    int64
	Device int64
	Seq    int64
	Action string
	R      func(message *message.Message)

	done bool
}

func (i *Context) Response(message *message.Message) {
	i.R(message)
}

func (i *Context) ReturnSuccess(data interface{}) {
	i.R(message.NewMessage(i.Seq, comm.ActionSuccess, data))
}

type IRoute interface {
	handle(path path, request *Context, data interface{}) error
}

type baseRoute struct {
	name   string
	parent *RtGroup
}

func (r *baseRoute) path() string {
	prefix := ""
	if r.parent != nil && r.parent.name != "" {
		prefix = r.parent.path() + PathSeparator
	}
	return prefix + r.name
}

type Rt struct {
	baseRoute
	handleFunc HandleFunc

	typeParam       reflect.Type
	shouldValidate  bool
	hasParam        bool
	valueHandleFunc reflect.Value
}

func (r *Rt) handle(_ path, request *Context, data interface{}) error {
	return r.invokeHandleFunc(request, data)
}

func (r *Rt) reflectHandleFunc() {
	typeHandleFunc := reflect.TypeOf(r.handleFunc)

	if typeHandleFunc.Kind() != reflect.Func {
		panic("the route handleFunc must be a function, route: " + r.name)
	}

	argNum := typeHandleFunc.NumIn()

	if argNum == 0 || argNum > 2 {
		panic("route handleFunc bad arguments, route: " + r.name)
	}

	// reflect request param
	if argNum == 2 {
		r.hasParam = true
		if typeHandleFunc.In(1).Kind() != reflect.Ptr {
			panic("route handleFunc param must be pointer, route: " + r.name)
		}

		r.typeParam = typeHandleFunc.In(1).Elem()
		if r.typeParam.Kind() != reflect.Struct {
			panic("the second arg of handleFunc must struct")
		}
		_, r.shouldValidate = reflect.New(r.typeParam).Interface().(Validatable)
	}

	// reflect first param
	if !typeHandleFunc.In(0).AssignableTo(typeRequestInfo) {
		panic("route handleFunc bad arguments, route: " + r.name)
	}

	r.valueHandleFunc = reflect.ValueOf(r.handleFunc)
}

func (r *Rt) invokeHandleFunc(info *Context, data interface{}) error {

	handleFuncArg := []interface{}{info}

	if r.hasParam {
		reqParam := reflect.New(r.typeParam).Interface()
		if r.shouldValidate {
			p := reqParam.(Validatable)
			err := p.Validate(data)
			if err != nil {
				// on validate request param failed
			}
			reqParam = reflect.ValueOf(p).Interface()
		} else {
			// TODO replace single json serializer as interface or other.
			m, ok := data.(*message.Message)
			if !ok {
				return errors.New("not type of *message.Message")
			}
			err := m.DeserializeData(reqParam)
			if err != nil {
				return err
			}
		}
		handleFuncArg = append(handleFuncArg, reqParam)
	}

	rt := r.valueHandleFunc.Call(valOf(handleFuncArg...))
	if len(rt) == 1 {
		err, ok := rt[0].Interface().(error)
		if ok {
			return err
		}
	}
	return nil
}

func (r *Rt) tryUnmarshal(i interface{}, jsonData interface{}) {
	s, ok := jsonData.(string)
	if ok {
		_ = json.Unmarshal([]byte(s), i)
	}
	bytes, ok := jsonData.([]byte)
	if ok {
		_ = json.Unmarshal(bytes, i)
	}
}

func valOf(i ...interface{}) []reflect.Value {
	var rt []reflect.Value
	for _, i2 := range i {
		rt = append(rt, reflect.ValueOf(i2))
	}
	return rt
}

func (r *Rt) String() string {
	return fmt.Sprintf("%s\t%v", r.path(), r.handleFunc)
}

type RtGroup struct {
	baseRoute
	rts map[string]IRoute
}

func (r *RtGroup) handle(path path, request *Context, data interface{}) error {
	p, b := path.next()
	if !b {
		return ErrNotRouteMatches
	}
	rt, ok := r.rts[p]
	if !ok {
		return ErrNotRouteMatches
	}
	return rt.handle(path, request, data)
}

func (r *RtGroup) Add(irt IRoute) {
	rt, ok := irt.(*Rt)
	if ok {
		rt.parent = r
		r.rts[rt.name] = rt
		return
	}
	g, ok := irt.(*RtGroup)
	if ok {
		g.parent = r
		r.rts[g.name] = g
	}
}

func (r *RtGroup) String() string {
	info := ""
	for _, route := range r.rts {
		rt, ok := route.(*Rt)
		if ok {
			info += "\n"
			info = info + rt.String()
		}
		rtGroup, ok := route.(*RtGroup)
		if ok {
			info += rtGroup.String()
		}
	}
	return info
}

type Router struct {
	root *RtGroup
}

func NewRouter() *Router {
	return &Router{root: Group("")}
}

func (r *Router) Add(rts ...IRoute) {
	for _, rt := range rts {
		r.root.Add(rt)
	}
}

func (r *Router) Handle(uid int64, device int64, msg *message.Message) (*message.Message, error) {
	ctx := &Context{
		Uid:    uid,
		Seq:    msg.GetSeq(),
		Device: device,
		Action: msg.GetAction(),
	}
	p := newPath(msg.GetAction())

	respCh := make(chan *message.Message)
	errCh := make(chan error)
	timeout := time.After(time.Second * 3)

	go func() {
		ctx.R = func(message *message.Message) {
			ctx.done = true
			respCh <- message
		}
		err := r.root.handle(p, ctx, msg)
		if err != nil && !ctx.done {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return nil, err
	case <-timeout:
		return nil, fmt.Errorf("timeout")
	case m := <-respCh:
		return m, nil
	}
}

func (r *Router) String() string {
	return r.root.String()
}

func Route(name string, handleFunc HandleFunc) IRoute {
	rt := &Rt{
		baseRoute:  baseRoute{name: name},
		handleFunc: handleFunc,
	}
	rt.reflectHandleFunc()
	return rt
}

func Group(name string, rts ...IRoute) *RtGroup {
	g := &RtGroup{
		baseRoute: baseRoute{name: name},
		rts:       make(map[string]IRoute),
	}

	for _, rt := range rts {
		g.Add(rt)
	}
	return g
}
