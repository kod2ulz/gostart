package utils

import (
	"bytes"
	"net/http"

	"github.com/gin-gonic/gin"
	json "github.com/json-iterator/go"
	"github.com/kod2ulz/gostart/collections"
	g "github.com/onsi/gomega"
)

type testUtils struct{}

var Test testUtils

func (testUtils) Request(method, path string, data []byte, headers ...map[string]string) (req *http.Request) {
	var err error
	var outBytes *bytes.Buffer
	if len(data) > 0 {
		outBytes = bytes.NewBuffer(data)
	}
	switch method {
	case http.MethodGet:
		req, err = http.NewRequest(method, path, nil)
	default:
		req, err = http.NewRequest(method, path, outBytes)
	}
	g.Expect(err).To(g.BeNil())
	g.Expect(req).ToNot(g.BeNil())
	req.Header.Set("Content-Type", "application/json")
	for _, h := range headers {
		for k, v := range h {
			req.Header.Set(k, v)
		}
	}
	return
}

func (testUtils) GinRouter(setRoutes func(*gin.Engine)) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	setRoutes(router)
	return router
}

func (testUtils) JsonData(in map[string]interface{}) (out []byte) {
	var err error
	out, err = json.Marshal(in)
	g.Expect(err).To(g.BeNil())
	return
}

func (testUtils) JsonEncode(in interface{}) (out []byte) {
	var err error
	out, err = json.Marshal(in)
	g.Expect(err).To(g.BeNil())
	return
}

func (u testUtils) JsonDataOf(val ...interface{}) (out []byte) {
	return u.JsonData(collections.MapOf[string, interface{}](val...))
}
