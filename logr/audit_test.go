package logr_test

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockAuditWriter is an in-memory writer for testing.
type MockAuditWriter struct {
	buffer *bytes.Buffer
}

func (m *MockAuditWriter) Write(entry map[string]interface{}) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = m.buffer.Write(append(line, '\n'))
	return err
}

var _ = Describe("AuditLogger", func() {

	BeforeEach(func() {
		// Ensure a clean logger setup for each test
		Expect(logr.Config()).To(Succeed())
	})

	Context("with a mock writer", func() {
		It("should write audit logs to the configured writer", func() {
			mockWriter := &MockAuditWriter{buffer: &bytes.Buffer{}}
			logr.SetAuditWriter(mockWriter)

			Expect(logr.Audit()).NotTo(BeNil())

			auditEntry := map[string]interface{}{
				"action":  "user_login",
				"user_id": "user-123",
			}
			logr.Audit().Log(auditEntry)

			var result map[string]interface{}
			Expect(json.Unmarshal(mockWriter.buffer.Bytes(), &result)).To(Succeed())
			Expect(result).To(HaveKeyWithValue("action", "user_login"))
			Expect(result).To(HaveKeyWithValue("user_id", "user-123"))
		})
	})

	Context("with a file writer", func() {
		It("should write audit logs to a file", func() {
			tempFile, err := os.CreateTemp("", "audit-test-*.log")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tempFile.Name())

			fileWriter, err := logr.NewFileAuditWriter(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			logr.SetAuditWriter(fileWriter)

			auditEntry := map[string]interface{}{"action": "file_audit"}
			logr.Audit().Log(auditEntry)

			// Close the writer to ensure flush
			if closer, ok := fileWriter.(interface{ Close() error }); ok {
				Expect(closer.Close()).To(Succeed())
			}

			content, err := os.ReadFile(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("file_audit"))
		})
	})
})
