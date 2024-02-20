package collections_test

import (
	"context"

	"github.com/google/uuid"
	colx "github.com/kod2ulz/gostart/collections"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TreeNode", func() {

	When("working with node trees", func() {
		var un *colx.TreeNode[uuid.UUID, Org]

		BeforeEach(func(ctx context.Context) {
			un = colx.TreeOf(unitedNations...)
			Expect(un.Size()).To(Equal(39))
		})

		AfterEach(func() { un.Clear() })

		It("can retrieve values from tree node", func() {
			var unhcr = unitedNations[5]                             // United Nations High Commissioner for Refugees
			var au = unitedNations[26]                               // African Union
			var eac = unitedNations[31]                              // East African Community
			Expect(un.Get(unhcr.ID).Value()).To(Equal(unhcr))        // can retrieve node from root by ID
			Expect(un.Get(eac.ID)).To(BeNil())                       // cannot get node not directly under another node
			Expect(un.Get(au.ID).Get(eac.ID).Value()).To(Equal(eac)) // an retrueve a node from under it's parent node
			Expect(un.Find(eac.ID).Value()).To(Equal(eac))           // can find any node from the root
			Expect(un.Find(uuid.New())).To(BeNil())                  // will find only nodes that exist
		})

		It("can measure distance (in hierarchy) between nodes", func() {
			var unhcr = unitedNations[5] // United Nations High Commissioner for Refugees
			var eac = unitedNations[31]  // East African Community
			dst, ok := un.Get(unhcr.ID).Distance(un.Get(eac.ID))
			Expect(ok).To(BeFalse())
			dst, ok = un.Get(unhcr.ID).Distance(un.Find(eac.ID))
			Expect(ok).To(BeTrue())
			Expect(dst).To(Equal(1))
		})

		It("can add node as child to another node", func() {
			var au = unitedNations[26]  // African Union
			var eac = unitedNations[31] // East African Community
			var uca = Org{ID: uuid.New(), Name: "UCASSC", ParentID: &eac.ID}
			affected := un.Find(eac.ID).Add(uca)
			Expect(affected).To(BeEquivalentTo(1))
			Expect(un.Find(uca.ID).Value()).To(Equal(uca))
			Expect(un.Get(au.ID).Get(eac.ID).Get(uca.ID).Value()).To(Equal(uca))
		})

		It("can remove node as child to another node", func() {
			var au = unitedNations[26]  // African Union
			var eac = unitedNations[31] // East African Community
			var uca = Org{ID: uuid.New(), Name: "UCASSD", ParentID: &eac.ID}
			Expect(un.Find(eac.ID).Add(uca)).To(Equal(int64(1)))
			ucaNode := un.Get(au.ID).Get(eac.ID).Get(uca.ID)
			Expect(ucaNode.Value()).To(Equal(uca))
			Expect(ucaNode.Remove()).To(Equal(int64(1)))
			Expect(un.Find(uca.ID)).To(BeNil())
			Expect(un.Get(au.ID).Get(eac.ID).Get(uca.ID)).To(BeNil())
		})
	})
})
