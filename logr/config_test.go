package logr_test

import (
	"github.com/kod2ulz/gostart/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {

	Context("when configuring the logger", func() {
		It("should initialize without error", func() {
			Expect(logr.Config()).To(Succeed())
		})

		It("should provide a non-nil logger instance", func() {
			Expect(logr.Config()).To(Succeed())
			Expect(logr.Log()).NotTo(BeNil())
		})
	})
})