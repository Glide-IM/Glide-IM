package json

import (
	"encoding/json"
	"go_im/pkg/logger"
)

type Data struct {
	des interface{}
}

func NewData(d interface{}) Data {
	return Data{
		des: d,
	}
}

func (d *Data) Data() interface{} {
	return d.des
}

func (d *Data) UnmarshalJSON(bytes []byte) error {
	d.des = bytes
	return nil
}

func (d *Data) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.des)
}

func (d *Data) bytes() []byte {
	bytes, ok := d.des.([]byte)
	if ok {
		return bytes
	}
	marshalJSON, err := d.MarshalJSON()
	if err != nil {
		logger.E("message data marshal json error %v", err)
		return nil
	}
	return marshalJSON
}

func (d *Data) Deserialize(i interface{}) error {
	s, ok := d.des.([]byte)
	if ok {
		return json.Unmarshal(s, i)
	}
	return nil
}

type ComMessage struct {
	Ver    int64
	Seq    int64
	Action string
	Data   Data
	Extra  map[string]string
}

func NewMessage(seq int64, action string, data interface{}) *ComMessage {
	return &ComMessage{
		Ver:    0,
		Seq:    seq,
		Action: action,
		Data:   NewData(data),
	}
}
