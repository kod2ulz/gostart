package logr_test

import (
	"bytes"
	"encoding/json"
	"os"
	"sync"

	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockAuditWriter is an in-memory writer for testing.
type MockAuditWriter struct {
	buffer *bytes.Buffer
	wg     *sync.WaitGroup
}

func (m *MockAuditWriter) Write(entry map[string]interface{}) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	_, err = m.buffer.Write(append(line, '\n'))
	m.wg.Done()
	return err
}

var _ = Describe("AuditLogger", func() {

	BeforeEach(func() {
		Expect(logr.Config()).To(Succeed())
	})

	Context("with a mock writer", func() {
	
		It("should write audit logs asynchronously", func() {
			wg := &sync.WaitGroup{}
			mockWriter := &MockAuditWriter{buffer: &bytes.Buffer{}, wg: wg}
			wg.Add(1)

			logr.SetAuditWriter(mockWriter)
			Expect(logr.Audit()).NotTo(BeNil())

			auditEntry := map[string]interface{}{
				"action":  "user_login",
				"user_id": "user-123",
			}
			logr.Audit().Log(auditEntry)

			// Wait for the async write to complete
			wg.Wait()

			var result map[string]interface{}
			Expect(json.Unmarshal(mockWriter.buffer.Bytes(), &result)).To(Succeed())
			Expect(result).To(HaveKeyWithValue("action", "user_login"))
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

			// In a real app, you'd have a central shutdown that closes loggers.
			// For this test, we can't easily access the internal non-blocking writer to close it,
			// so we'll just wait a moment for the async write to likely complete.
			Eventually(func() string {
				content, _ := os.ReadFile(tempFile.Name())
				return string(content)
			}).Should(ContainSubstring("file_audit"))
		})
	})
})