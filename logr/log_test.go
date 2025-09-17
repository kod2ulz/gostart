package logr_test

import (
	"bytes"
	"encoding/json"
	"log/slog"

	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logger", func() {

	var (
		buffer *bytes.Buffer
	)

	BeforeEach(func() {
		buffer = &bytes.Buffer{}
		handler := slog.NewJSONHandler(buffer, &slog.HandlerOptions{
			AddSource: true,
		})
		logger := slog.New(handler)
		logr.SetUpLogger(logger, nil)
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

		It("should add a custom field with ExtendWithField()", func() {
			logr.Log().ExtendWithField("request_url", "/my/path").Info("test message")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKeyWithValue("request_url", "/my/path"))
		})

		It("should chain multiple fields correctly", func() {
			logr.Log().TID().PID().ExtendWithField("request_url", "/path").Info("chained test")
			var logOutput map[string]interface{}
			Expect(json.Unmarshal(buffer.Bytes(), &logOutput)).To(Succeed())
			Expect(logOutput).To(HaveKey("trace_id"))
			Expect(logOutput).To(HaveKey("process_id"))
			Expect(logOutput).To(HaveKeyWithValue("request_url", "/path"))
		})
	})
})
