package logr_test

import (
	"os"

	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Context("when configuring the logger", func() {
		It("should initialize with default console handler without error", func() {
			Expect(logr.Config()).To(Succeed())
			Expect(logr.Log()).NotTo(BeNil())
		})

		It("should initialize with a pretty console handler", func() {
			consoleHandler := logr.NewConsoleHandler(true)
			Expect(logr.Config(consoleHandler)).To(Succeed())
			Expect(logr.Log()).NotTo(BeNil())
		})

		It("should initialize with a file handler", func() {
			tempFile, err := os.CreateTemp("", "test-*.log")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tempFile.Name())

			fileHandler := logr.NewFileHandler(tempFile.Name(), nil)
			Expect(logr.Config(fileHandler)).To(Succeed())
			Expect(logr.Log()).NotTo(BeNil())

			// Log a message and check if it's in the file
			logr.Log().Info("hello from file logger")
			content, err := os.ReadFile(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("hello from file logger"))
		})

		It("should initialize with multiple handlers", func() {
			consoleHandler := logr.NewConsoleHandler(false)

			tempFile, err := os.CreateTemp("", "multi-*.log")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tempFile.Name())
			fileHandler := logr.NewFileHandler(tempFile.Name(), nil)

			Expect(logr.Config(consoleHandler, fileHandler)).To(Succeed())
			Expect(logr.Log()).NotTo(BeNil())

			// Log a message and check if it's in the file
			logr.Log().Warn("this should go to both handlers")
			content, err := os.ReadFile(tempFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("this should go to both handlers"))
		})
	})
})
