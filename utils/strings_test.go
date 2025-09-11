
package utils_test

import (
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("String Utils", func() {

	Context("Numeric conversions", func() {
		It("should convert a string to an int", func() {
			Expect(utils.String.ToInt("42")).To(Equal(42))
			Expect(utils.String.ToInt("-10")).To(Equal(-10))
			Expect(utils.String.ToInt("not-a-number")).To(BeZero())
		})

		It("should convert a string to an int64", func() {
			Expect(utils.String.ToInt64("1234567890")).To(Equal(int64(1234567890)))
			Expect(utils.String.ToInt64("not-a-number")).To(BeZero())
		})

		It("should convert a slice of strings to a slice of int64s", func() {
			input := []string{"1", "2", "not-a-number", "4"}
			expected := []int64{1, 2, 0, 4}
			Expect(utils.String.ToInt64Slice(input)).To(Equal(expected))
			Expect(utils.String.ToInt64Slice([]string{})).To(BeEmpty())
		})

		It("should convert a string to a float64", func() {
			Expect(utils.String.ToFloat64("123.45")).To(Equal(123.45))
			Expect(utils.String.ToFloat64("not-a-number")).To(BeZero())
		})
	})

	Context("UUID conversions", func() {
		It("should convert a valid string to a UUID", func() {
			uid := uuid.New()
			Expect(utils.String.ToUUID(uid.String())).To(Equal(uid))
		})

		It("should return a nil UUID for an invalid string", func() {
			Expect(utils.String.ToUUID("not-a-uuid")).To(Equal(uuid.Nil))
		})

		It("should convert a valid string to a NullUUID", func() {
			uid := uuid.New()
			nullUUID := utils.String.ToNullUUID(uid.String())
			Expect(nullUUID.Valid).To(BeTrue())
			Expect(nullUUID.UUID).To(Equal(uid))
		})

		It("should return an invalid NullUUID for an invalid string", func() {
			Expect(utils.String.ToNullUUID("not-a-uuid").Valid).To(BeFalse())
		})
	})

	Context("String manipulation", func() {
		It("should trim prefixes from a string", func() {
			Expect(utils.String.TrimPrefixes("__my_string__", "__")).To(Equal("my_string__"))
			Expect(utils.String.TrimPrefixes("abchello", "a", "b")).To(Equal("chello"))
			Expect(utils.String.TrimPrefixes("hello", "a", "b")).To(Equal("hello"))
		})

		It("should generate a random string of a given length", func() {
			Expect(len(utils.String.Random(10))).To(Equal(10))
			Expect(len(utils.String.Random(0))).To(Equal(0))
		})

		It("should omit empty strings from a slice", func() {
			input := []string{"a", "", "  ", "b", "c "}
			expected := []string{"a", "b", "c"}
			Expect(utils.String.OmitEmpty(input...)).To(Equal(expected))
			Expect(utils.String.OmitEmpty()).To(BeEmpty())
		})

		It("should join non-empty strings", func() {
			Expect(utils.String.Join(",", "a", "b", "", " c ")).To(Equal("a,b,c"))
			Expect(utils.String.Join(",", "a")).To(Equal("a"))
			Expect(utils.String.Join(",", "", " ")).To(BeEmpty())
		})
	})
})
