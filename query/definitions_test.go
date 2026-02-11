package query_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/query"
)

var _ = Describe("Field Definitions", func() {

	var defs query.FieldDefinitions

	BeforeEach(func() {
		defs = query.NewDefinitions()
	})

	Describe("NewDefinitions", func() {
		It("should create empty FieldDefinitions", func() {
			Expect(defs).NotTo(BeNil())
			Expect(len(defs)).To(Equal(0))
		})

		It("should create FieldDefinitions with initial fields", func() {
			defs = query.NewDefinitions(
				query.Text("name"),
				query.Int("age"),
			)
			Expect(len(defs)).To(Equal(2))
			Expect(defs["name"].Name).To(Equal("name"))
			Expect(defs["age"].Name).To(Equal("age"))
		})
	})

	Describe("Field Creation Functions", func() {
		It("should create Text fields with correct defaults", func() {
			field := query.Text("name")
			Expect(field.Name).To(Equal("name"))
			Expect(field.DBName).To(Equal("name"))
			Expect(field.Type).To(Equal(query.TypeText))
			Expect(field.Sort).To(BeFalse()) // Default from newField
			Expect(field.Operators).To(ContainElements(
				query.CompareEqual,
				query.CompareNotEqual,
				query.CompareIn,
				query.CompareLike,
			))
		})

		It("should create Int fields with correct defaults", func() {
			field := query.Int("age")
			Expect(field.Name).To(Equal("age"))
			Expect(field.DBName).To(Equal("age"))
			Expect(field.Type).To(Equal(query.TypeInt))
			Expect(field.Operators).To(ContainElements(
				query.CompareEqual,
				query.CompareNotEqual,
				query.CompareIn,
				query.CompareBetween,
				query.CompareGreaterThan,
				query.CompareGreaterThanOrEqual,
				query.CompareLessThan,
				query.CompareLessThanOrEqual,
			))
		})

		It("should create Float fields with correct defaults", func() {
			field := query.Float("score")
			Expect(field.Name).To(Equal("score"))
			Expect(field.DBName).To(Equal("score"))
			Expect(field.Type).To(Equal(query.TypeFloat))
			Expect(field.Operators).To(ContainElements(
				query.CompareEqual,
				query.CompareNotEqual,
				query.CompareIn,
				query.CompareBetween,
				query.CompareGreaterThan,
				query.CompareGreaterThanOrEqual,
				query.CompareLessThan,
				query.CompareLessThanOrEqual,
			))
		})

		It("should create Date fields with correct defaults", func() {
			field := query.Date("birth_date", "2006-01-02")
			Expect(field.Name).To(Equal("birth_date"))
			Expect(field.DBName).To(Equal("birth_date"))
			Expect(field.Type).To(Equal(query.TypeDate))
			Expect(field.TimeFormat).To(Equal("2006-01-02"))
		})

		It("should create Date fields without format", func() {
			field := query.Date("created_at")
			Expect(field.Name).To(Equal("created_at"))
			Expect(field.DBName).To(Equal("created_at"))
			Expect(field.Type).To(Equal(query.TypeDate))
			Expect(field.TimeFormat).To(BeEmpty())
		})

		It("should create Time fields with correct defaults", func() {
			field := query.Time("updated_at", "2006-01-02 15:04:05")
			Expect(field.Name).To(Equal("updated_at"))
			Expect(field.DBName).To(Equal("updated_at"))
			Expect(field.Type).To(Equal(query.TypeTime))
			Expect(field.TimeFormat).To(Equal("2006-01-02 15:04:05"))
		})

		It("should create UUID fields with correct defaults", func() {
			field := query.UUID("id")
			Expect(field.Name).To(Equal("id"))
			Expect(field.DBName).To(Equal("id"))
			Expect(field.Type).To(Equal(query.TypeUUID))
			Expect(field.Operators).To(ContainElements(
				query.CompareEqual,
				query.CompareNotEqual,
				query.CompareIn,
			))
		})

		It("should create Bool fields with correct defaults", func() {
			field := query.Bool("active")
			Expect(field.Name).To(Equal("active"))
			Expect(field.DBName).To(Equal("active"))
			Expect(field.Type).To(Equal(query.TypeBool))
			Expect(field.Operators).To(ContainElements(
				query.CompareEqual,
				query.CompareNotEqual,
			))
		})

		It("should create JSON fields with correct defaults", func() {
			field := query.JSON("metadata")
			Expect(field.Name).To(Equal("metadata"))
			Expect(field.DBName).To(Equal("metadata"))
			Expect(field.Type).To(Equal(query.TypeJSON))
			Expect(field.Schema).To(BeNil())
		})
	})

	Describe("Field Definition Methods", func() {
		var field query.FieldDefinition

		BeforeEach(func() {
			field = query.Text("name")
		})

		Describe("WithDBName", func() {
			It("should override the default database column name", func() {
				updated := field.WithDBName("full_name")
				Expect(updated.DBName).To(Equal("full_name"))
				Expect(updated.Name).To(Equal("name")) // Original name unchanged
			})
		})

		Describe("Sortable", func() {
			It("should enable sorting when called without arguments", func() {
				updated := field.Sortable()
				Expect(updated.Sort).To(BeTrue())
			})

			It("should enable sorting when called with true", func() {
				updated := field.Sortable(true)
				Expect(updated.Sort).To(BeTrue())
			})

			It("should disable sorting when called with false", func() {
				updated := field.Sortable(false)
				Expect(updated.Sort).To(BeFalse())
			})
		})

		Describe("WithOperators", func() {
			It("should set the list of allowed comparison operators", func() {
				updated := field.WithOperators(query.CompareEqual, query.CompareLike)
				Expect(updated.Operators).To(ContainElements(query.CompareEqual, query.CompareLike))
				Expect(len(updated.Operators)).To(Equal(2))
			})

			It("should clear operators when called with no arguments", func() {
				updated := field.WithOperators()
				Expect(len(updated.Operators)).To(Equal(0))
			})
		})

		Describe("WithSchema", func() {
			It("should define nested schema for JSON fields", func() {
				jsonField := query.JSON("metadata")
				updated := jsonField.WithSchema(
					query.Text("city"),
					query.Int("age"),
				)

				Expect(updated.Schema).NotTo(BeNil())
				Expect(len(updated.Schema)).To(Equal(2))
				Expect(updated.Schema["city"].Name).To(Equal("city"))
				Expect(updated.Schema["city"].Type).To(Equal(query.TypeText))
				Expect(updated.Schema["age"].Name).To(Equal("age"))
				Expect(updated.Schema["age"].Type).To(Equal(query.TypeInt))
			})

			It("should ignore schema for non-JSON fields", func() {
				textField := query.Text("name")
				updated := textField.WithSchema(query.Text("first"), query.Text("last"))
				Expect(updated.Schema).To(BeNil()) // Should be ignored
			})

			It("should add to existing schema", func() {
				jsonField := query.JSON("metadata").WithSchema(query.Text("city"))
				updated := jsonField.WithSchema(query.Int("age"))

				Expect(len(updated.Schema)).To(Equal(2))
				Expect(updated.Schema["city"]).NotTo(BeNil())
				Expect(updated.Schema["age"]).NotTo(BeNil())
			})
		})
	})

	Describe("FieldDefinitions Methods", func() {
		BeforeEach(func() {
			defs = query.NewDefinitions()
		})

		Describe("Add", func() {
			It("should add single field to definitions", func() {
				field := query.Text("name")
				updated := defs.Add(field)

				Expect(len(updated)).To(Equal(1))
				Expect(updated["name"]).To(Equal(field))
			})

			It("should add multiple fields to definitions", func() {
				updated := defs.Add(
					query.Text("name"),
					query.Int("age"),
					query.Float("score"),
				)

				Expect(len(updated)).To(Equal(3))
				Expect(updated["name"].Type).To(Equal(query.TypeText))
				Expect(updated["age"].Type).To(Equal(query.TypeInt))
				Expect(updated["score"].Type).To(Equal(query.TypeFloat))
			})

			It("should overwrite existing field with same name", func() {
				initial := defs.Add(query.Text("name"))
				Expect(initial["name"].Type).To(Equal(query.TypeText))

				updated := initial.Add(query.Int("name"))
				Expect(len(updated)).To(Equal(1))
				Expect(updated["name"].Type).To(Equal(query.TypeInt))
			})

			It("should return the same map when adding no fields", func() {
				updated := defs.Add()
				Expect(updated).To(Equal(defs))
			})
		})
	})

	Describe("Practical Usage Examples", func() {
		It("should create comprehensive user field definitions", func() {
			userDefs := query.NewDefinitions(
				query.UUID("id").Sortable(false),
				query.Text("username").Sortable().WithOperators(
					query.CompareEqual, query.CompareLike, query.CompareIn,
				),
				query.Text("email").WithOperators(query.CompareEqual, query.CompareIn),
				query.Int("age").WithOperators(
					query.CompareEqual, query.CompareGreaterThan,
					query.CompareLessThan, query.CompareBetween,
				),
				query.Float("score").Sortable(),
				query.Bool("active"),
				query.Date("created_at", time.RFC3339).Sortable(),
				query.JSON("preferences").WithSchema(
					query.Text("theme"),
					query.Text("language"),
					query.Bool("notifications"),
				),
			)

			Expect(len(userDefs)).To(Equal(8))

			// Test field properties
			Expect(userDefs["username"].Sort).To(BeTrue())
			Expect(userDefs["id"].Sort).To(BeFalse())
			Expect(userDefs["username"].Operators).To(ContainElements(query.CompareLike))
			Expect(userDefs["preferences"].Type).To(Equal(query.TypeJSON))
			Expect(len(userDefs["preferences"].Schema)).To(Equal(3))
		})

		It("should create product catalog field definitions", func() {
			productDefs := query.NewDefinitions(
				query.UUID("id"),
				query.Text("name").Sortable().WithOperators(
					query.CompareEqual, query.CompareLike, query.CompareIn,
				),
				query.Text("description").WithOperators(query.CompareLike),
				query.Float("price").WithOperators(
					query.CompareEqual, query.CompareGreaterThan,
					query.CompareLessThan, query.CompareBetween,
				),
				query.Int("stock_quantity"),
				query.Bool("in_stock"),
				query.JSON("attributes").WithSchema(
					query.Text("color"),
					query.Text("size"),
					query.Float("weight"),
					query.JSON("dimensions").WithSchema(
						query.Float("length"),
						query.Float("width"),
						query.Float("height"),
					),
				),
				query.Date("created_at", time.RFC3339),
				query.Date("updated_at", time.RFC3339),
			)

			Expect(len(productDefs)).To(Equal(9))

			// Test nested JSON schema
			attributes := productDefs["attributes"].Schema
			Expect(attributes).NotTo(BeNil())
			Expect(len(attributes)).To(Equal(4)) // color, size, weight, dimensions

			dimensions := attributes["dimensions"].Schema
			Expect(dimensions).NotTo(BeNil())
			Expect(len(dimensions)).To(Equal(3)) // length, width, height
		})
	})

	Describe("Field Validation", func() {
		It("should handle complex operator combinations", func() {
			field := query.Text("search_field").WithOperators(
				query.CompareEqual,
				query.CompareNotEqual,
				query.CompareIn,
				query.CompareLike,
				query.CompareAny,
				query.CompareExists,
			)

			Expect(len(field.Operators)).To(Equal(6))
			Expect(field.Operators).To(ContainElements(
				query.CompareEqual,
				query.CompareLike,
				query.CompareAny,
				query.CompareExists,
			))
		})

		It("should maintain field immutability when using builder methods", func() {
			original := query.Text("name")
			modified := original.Sortable(true).WithDBName("full_name")

			// Original should be unchanged
			Expect(original.Sort).To(BeFalse())
			Expect(original.DBName).To(Equal("name"))

			// Modified should have changes
			Expect(modified.Sort).To(BeTrue())
			Expect(modified.DBName).To(Equal("full_name"))
		})

		It("should support chain operations", func() {
			field := query.JSON("metadata").
				WithDBName("user_metadata").
				WithSchema(
					query.Text("city").Sortable(),
					query.Int("age").WithOperators(
						query.CompareEqual,
						query.CompareGreaterThan,
						query.CompareLessThan,
					),
				)

			Expect(field.DBName).To(Equal("user_metadata"))
			Expect(len(field.Schema)).To(Equal(2))
			Expect(field.Schema["city"].Sort).To(BeTrue())
			Expect(len(field.Schema["age"].Operators)).To(Equal(3))
		})
	})
})
