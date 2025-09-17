package object_test

import (
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/object"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("String Object", func() {

	Context("Split method", func() {
		It("should split a string by a separator", func() {
			s := object.String("a,b,c")
			Expect(s.Split(",")).To(Equal(collections.List[string]{"a", "b", "c"}))
		})

		It("should return the original string if separator is not found", func() {
			s := object.String("abc")
			Expect(s.Split(",")).To(Equal(collections.List[string]{"abc"}))
		})

		It("should return an empty slice for an empty string", func() {
			s := object.String("")
			Expect(s.Split(",")).To(BeEmpty())
		})
	})

	Context("Variations method", func() {
		It("should create variations of a string using formats", func() {
			s := object.String("test")
			formats := []string{"prefix-%s", "%s-suffix", "pre-%s-suf"}
			Expect(s.Variations(formats...)).To(Equal(collections.List[string]{"prefix-test", "test-suffix", "pre-test-suf"}))
		})

		It("should return the original string if no formats are given", func() {
			s := object.String("test")
			Expect(s.Variations()).To(Equal(collections.List[string]{"test"}))
		})
	})

	Context("SubstringBefore method", func() {
		It("should return the substring before the given character", func() {
			s := object.String("hello-world")
			Expect(s.SubstringBefore('-').String()).To(Equal("hello"))
		})

		It("should return the original string if the character is not found", func() {
			s := object.String("helloworld")
			Expect(s.SubstringBefore('-').String()).To(Equal("helloworld"))
		})

		It("should handle an empty string", func() {
			s := object.String("")
			Expect(s.SubstringBefore('-').String()).To(BeEmpty())
		})
	})
})
