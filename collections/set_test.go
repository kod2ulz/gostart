package collections_test

import (
	"github.com/kod2ulz/gostart/collections"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Set", func() {
	var set1, set2 collections.Set[string]

	BeforeEach(func() {
		set1 = collections.Set[string]{}
		set1.Add("apple")
		set1.Add("banana")
		set1.Add("cherry")

		set2 = collections.Set[string]{}
		set2.Add("cherry")
		set2.Add("durian")
	})

	It("should add and check for items", func() {
		Expect(set1.Has("apple")).To(BeTrue())
		Expect(set1.Has("durian")).To(BeFalse())
		Expect(set1.Size()).To(Equal(3))
	})

	It("should remove an item", func() {
		set1.Remove("banana")
		Expect(set1.Has("banana")).To(BeFalse())
		Expect(set1.Size()).To(Equal(2))
	})

	It("should perform a union", func() {
		union := set1.Union(set2)
		Expect(union.Size()).To(Equal(4))
		Expect(union.Has("apple")).To(BeTrue())
		Expect(union.Has("banana")).To(BeTrue())
		Expect(union.Has("cherry")).To(BeTrue())
		Expect(union.Has("durian")).To(BeTrue())
	})

	It("should perform an intersection", func() {
		intersection := set1.Intersection(set2)
		Expect(intersection.Size()).To(Equal(1))
		Expect(intersection.Has("cherry")).To(BeTrue())
		Expect(intersection.Has("apple")).To(BeFalse())
	})

	It("should perform a difference", func() {
		difference := set1.Difference(set2) // Items in set1 but not in set2
		Expect(difference.Size()).To(Equal(2))
		Expect(difference.Has("apple")).To(BeTrue())
		Expect(difference.Has("banana")).To(BeTrue())
		Expect(difference.Has("cherry")).To(BeFalse())
	})
})
