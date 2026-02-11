package utils_test

import (
	"github.com/kod2ulz/gostart/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Numeric Utils", func() {

	Context("Max function", func() {
		It("should find the maximum of positive integers", func() {
			Expect(utils.Max(1, 5, 3, 2)).To(Equal(5))
		})
		It("should find the maximum of negative integers", func() {
			Expect(utils.Max(-1, -5, -3, -2)).To(Equal(-1))
		})
		It("should handle a single number", func() {
			Expect(utils.Max(42)).To(Equal(42))
		})
		It("should return zero for an empty slice", func() {
			Expect(utils.Max[int]()).To(BeZero())
		})
	})

	Context("Min function", func() {
		It("should find the minimum of positive integers", func() {
			Expect(utils.Min(5, 1, 3, 2)).To(Equal(1))
		})
		It("should find the minimum of negative integers", func() {
			Expect(utils.Min(-5, -1, -3, -2)).To(Equal(-5))
		})
		It("should handle a single number", func() {
			Expect(utils.Min(42)).To(Equal(42))
		})
		It("should return zero for an empty slice", func() {
			Expect(utils.Min[int]()).To(BeZero())
		})
	})

	Context("Round function", func() {
		It("should round a float32 to the given precision", func() {
			Expect(utils.Round(2, float32(123.456))).To(Equal(float32(123.46)))
			Expect(utils.Round(1, float32(123.449))).To(Equal(float32(123.4)))
		})
		It("should round a float64 to the given precision", func() {
			Expect(utils.Round(2, 123.456)).To(Equal(123.46))
			Expect(utils.Round(0, 123.500)).To(Equal(124.0))
		})
	})

	Context("FormatMoney function", func() {
		It("should format the money correctly", func() {
			Expect(utils.FormatMoney("USD", 12345.678, 2)).To(Equal("USD 12,345.68"))
			Expect(utils.FormatMoney("UGX", 1000000, 0)).To(Equal("UGX 1,000,000.00")) // Note: Sprintf default is %.2f
		})
	})
})
