package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestApi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Api Suite")
}

func jsonFromObj(in interface{}) (out []byte) {
	var err error
	out, err = json.Marshal(in)
	Expect(err).To(BeNil())
	return
}

func jsonData(in map[string]interface{}) (out []byte) {
	var err error
	out, err = json.Marshal(in)
	Expect(err).To(BeNil())
	return
}

func jsonDataOf(val ...interface{}) (out []byte) {
	return jsonData(mapOf(val...))
}

func makeRequest(method, path string, data []byte) (req *http.Request) {
	var err error
	if len(data) > 0 {
		req, err = http.NewRequest(method, path, bytes.NewBuffer(data))
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	Expect(err).To(BeNil())
	Expect(req).ToNot(BeNil())
	req.Header.Set("Content-Type", "application/json")
	return
}

var mapOf = collections.MapOf[string, interface{}]

type ResultModel[P api.RequestParam, R any] map[string]interface{}

func (e ResultModel[P, R]) HasError() (yes bool) {
	if len(e) == 0 {
		return
	}
	_, yes = e["error"]
	return
}

func (e ResultModel[P, R]) Error() (er api.ErrorModel[P]) {
	if e.HasError() {
		utils.StructCopy(e["error"], &er)
		return 
	}
	return
}

func (e ResultModel[P, R]) Data() (out R) {
	utils.StructCopy(e["data"], &out)
	return
}

func (e ResultModel[P, R]) Parse(out interface{}) (err error) {
	return utils.StructCopy(e, out)
}
