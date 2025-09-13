package utils

import (
	"fmt"

	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/logr"
	json "github.com/json-iterator/go"
)

type jsonUtils struct{}

var JSON jsonUtils

func (u jsonUtils) PrettyString(log *logr.Logger, obj interface{}) string {
	if data, err := json.MarshalIndent(obj, "", "  "); err != nil {
		log.Error("data marshalling failed", "object_type", fmt.Sprintf("%T", obj), "error", err)
	} else {
		return string(data)
	}
	return ""
}

func (u jsonUtils) Bytes(log *logr.Logger, obj interface{}) (data []byte) {
	var err error
	if data, err = json.Marshal(obj); err != nil {
		log.Error("data marshalling failed", "object_type", fmt.Sprintf("%T", obj), "error", err)
	}
	return
}

func (u jsonUtils) String(log *logr.Logger, obj interface{}) string {
	if data, err := json.Marshal(obj); err != nil {
		log.Error("data marshalling failed", "object_type", fmt.Sprintf("%T", obj), "error", err)
	} else {
		return string(data)
	}
	return ""
}

func (u jsonUtils) Decode(log *logr.Logger, data []byte, obj interface{}) (interface{}) {
	if err := json.Unmarshal(data, &obj); err != nil {
		log.Error("data un-marshalling failed", "object_type", fmt.Sprintf("%T", obj), "error", err)
	}
	return obj
}

func (u jsonUtils) ToMap(obj interface{}) (out collections.Map[string, interface{}]) {
	data, err := json.Marshal(obj)
	if err == nil {
		json.Unmarshal(data, &obj)
	}
	return
}
