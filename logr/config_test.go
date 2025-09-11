
package logr_test

import (
	"os"

	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Context("when configuring file logging", func() {
		var (
			tempFile *os.File
			err      error
		)

		BeforeEach(func() {
			// Create a temporary file for logging
			tempFile, err = os.CreateTemp("", "log-test-*.log")
			Expect(err).NotTo(HaveOccurred())
			os.Setenv(logr.ENV_LOG_FILE_PATH, tempFile.Name())
		})

		AfterEach(func() {
			// Clean up the environment variable and temporary file
			os.Unsetenv(logr.ENV_LOG_FILE_PATH)
			if tempFile != nil {
				tempFile.Close()
				os.Remove(tempFile.Name())
			}
		})

		It("should write log messages to the specified file", func() {
			// Configure the logger, which should now pick up the file path
			Expect(logr.Config()).To(Succeed())

			// Log a unique message
			logMessage := "this is a test message for file logging"
			logr.Log().Info(logMessage)

			// Read the content of the file
			content, err := os.ReadFile(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())

			// Check if the log message is in the file content
			Expect(string(content)).To(ContainSubstring(logMessage))
		})
	})
})
