package utils

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kod2ulz/gostart/collections"
	g "github.com/onsi/gomega"
)

type testUtils struct{}

var Test testUtils

func (testUtils) Request(method, path string, data []byte) (req *http.Request) {
	var err error
	if len(data) > 0 {
		req, err = http.NewRequest(method, path, bytes.NewBuffer(data))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	g.Expect(err).To(g.BeNil())
	g.Expect(req).ToNot(g.BeNil())
	req.Header.Set("Content-Type", "application/json")
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