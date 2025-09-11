
package logr_test

import (
	"bytes"
	"encoding/json"

	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Logger", func() {

	var (
		buffer *bytes.Buffer
	)

	BeforeEach(func() {
		buffer = &bytes.Buffer{}

		// Initialize the global logger
		entry := logrus.NewEntry(logrus.New())
		entry.Logger.SetOutput(buffer)
		logr.SetUpLogger(entry)
		logr.SetFormatterJSON() // Ensure predictable JSON output for tests
	})

	Context("when adding context fields", func() {

		It("should add a Trace ID with TID()", func() {
			logr.Log().TID().Info("test message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKey("trace_id"))
			Expect(logOutput["trace_id"]).To(Not(BeEmpty()))
		})

		It("should add a specific Trace ID with WithTID()", func() {
			logr.Log().WithTID("my-trace-id").Info("test message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKeyWithValue("trace_id", "my-trace-id"))
		})

		It("should add a Process ID with PID()", func() {
			logr.Log().PID().Info("test message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKey("process_id"))
			Expect(logOutput["process_id"]).To(Not(BeEmpty()))
		})

		It("should add an incoming request URL", func() {
			logr.Log().InReqURL("/my/path").Info("test message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKeyWithValue("income_request_url", "/my/path"))
		})

		It("should add an outgoing request URL", func() {
			logr.Log().OutReqURL("http://example.com").Info("test message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKeyWithValue("outcome_request_url", "http://example.com"))
		})

		It("should add a full message", func() {
			logr.Log().FMsg("my full message").Info("test short message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKeyWithValue("full_message", "my full message"))
		})

		It("should chain multiple fields correctly", func() {
			logr.Log().TID().PID().InReqURL("/path").Info("chained test")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKey("trace_id"))
			Expect(logOutput).To(HaveKey("process_id"))
			Expect(logOutput).To(HaveKeyWithValue("income_request_url", "/path"))
		})
	})
})
