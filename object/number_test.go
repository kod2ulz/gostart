
package object_test

import (
	"github.com/kod2ulz/gostart/object"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Number Object", func() {

	Context("Num function", func() {
		It("should convert a string to an int", func() {
			val := object.Num[int]("12345")
			Expect(val).To(BeAssignableToTypeOf(0))
			Expect(val).To(Equal(12345))
		})

		It("should convert a string to an int64", func() {
			val := object.Num[int64]("9876543210")
			Expect(val).To(BeAssignableToTypeOf(int64(0)))
			Expect(val).To(Equal(int64(9876543210)))
		})

		It("should return 0 for a non-numeric string", func() {
			val := object.Num[int]("not-a-number")
			Expect(val).To(BeZero())
		})

		It("should handle negative numbers", func() {
			val := object.Num[int]("-50")
			Expect(val).To(Equal(-50))
		})
	})
})
