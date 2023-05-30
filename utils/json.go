package utils

import (

	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/logr"
	"github.com/sirupsen/logrus"
	json "github.com/json-iterator/go"
)

type jsonUtils struct{}

var JSON jsonUtils

func (u jsonUtils) PrettyString(log *logr.Logger, obj interface{}) string {
	if data, err := json.MarshalIndent(obj, "", "  "); err != nil {
		log.Printf("%T: data marshalling of %T failed. %v", u, obj, err)
	} else {
		return string(data)
	}
	return ""
}

func (u jsonUtils) Bytes(log *logr.Logger, obj interface{}) (data []byte) {
	var err error
	if data, err = json.Marshal(obj); err != nil {
		log.Printf("data marshalling of %T failed. %v", obj, err)
	}
	return
}

func (u jsonUtils) String(log *logrus.Entry, obj interface{}) string {
	if data, err := json.Marshal(obj); err != nil {
		log.Printf("data marshalling of %T failed. %v", obj, err)
	} else {
		return string(data)
	}
	return ""
}

func (u jsonUtils) Decode(log *logrus.Entry, data []byte) (obj interface{}) {
	if err := json.Unmarshal(data, obj); err != nil {
		log.Printf("data un marshalling of %T failed. %v", obj, err)
	}
	return
}

func (u jsonUtils) ToMap(obj interface{}) (out collections.Map[string, interface{}]) {
	data, err := json.Marshal(obj)
	if err == nil {
		json.Unmarshal(data, &obj)
	}
	return
}
