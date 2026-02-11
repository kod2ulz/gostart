package errors_test

import (
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type TestValidationRequest struct {
	ID       string `json:"id" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"gte=0"`
	Password string `json:"password" validate:"required,min=8"`
}

var _ = Describe("Validation Error Handling", func() {

	Describe("Multiple Validation Errors", func() {
		It("should return all validation errors at once", func() {
			req := TestValidationRequest{
				// All fields are missing/invalid
			}

			err := utils.Validate.Struct(req)
			Expect(err).To(HaveOccurred())

			validationErr := errors.ValidatorError[TestValidationRequest](err)
			Expect(validationErr).NotTo(BeNil())

			// Type assert to *errors.ErrorModel[TestValidationRequest] to access Fields
			if errorModel, ok := validationErr.(*errors.ErrorModel[TestValidationRequest]); ok {
				fields := errorModel.Fields
				// Should have multiple validation errors
				Expect(len(fields)).To(BeNumerically(">", 1),
					"Should return multiple validation errors at once")

				// Verify field names use JSON tags (camelCase) not Go struct names (PascalCase)
				Expect(fields).To(HaveKey("id"), "Should use JSON tag name 'id' not 'ID'")
				Expect(fields).To(HaveKey("name"), "Should use JSON tag name 'name' not 'Name'")
				Expect(fields).To(HaveKey("email"), "Should use JSON tag name 'email' not 'Email'")
			}
		})

		It("should use JSON tag names in error messages", func() {
			req := TestValidationRequest{}

			err := utils.Validate.Struct(req)
			Expect(err).To(HaveOccurred())

			validationErr := errors.ValidatorError[TestValidationRequest](err)
			Expect(validationErr).NotTo(BeNil())

			// The error message should mention the JSON tag names
			errMsg := validationErr.Error()
			Expect(errMsg).To(Or(
				ContainSubstring("id"),
				ContainSubstring("name"),
				ContainSubstring("email"),
			), "Error message should use JSON tag names (camelCase)")

			// Should NOT contain PascalCase field names
			Expect(errMsg).NotTo(Or(
				ContainSubstring("ID"),
				ContainSubstring("Name"),
				ContainSubstring("Email"),
			), "Error message should NOT use Go struct field names (PascalCase)")
		})

		It("should extract fields from Response map in ParseValidatorError", func() {
			req := TestValidationRequest{
				// All fields are missing/invalid
			}

			err := utils.Validate.Struct(req)
			Expect(err).To(HaveOccurred())

			validationErr := errors.ValidatorError[TestValidationRequest](err)
			Expect(validationErr).NotTo(BeNil())

			// This simulates what happens in HandleAPIError -> HandleValidationError -> ParseValidatorError
			validatorInfo := errors.ParseValidatorError(validationErr)
			Expect(validatorInfo).NotTo(BeNil(), "ParseValidatorError should extract validation info")
			Expect(validatorInfo.Fields).NotTo(BeNil(), "Should have fields map")
			Expect(len(validatorInfo.Fields)).To(BeNumerically(">", 1),
				"Should have multiple validation errors in fields")

			// Verify field names use JSON tags (camelCase)
			Expect(validatorInfo.Fields).To(HaveKey("id"))
			Expect(validatorInfo.Fields).To(HaveKey("name"))
			Expect(validatorInfo.Fields).To(HaveKey("email"))
		})

		It("should populate fields in HandleAPIError flow", func() {
			req := TestValidationRequest{
				// All fields are missing/invalid
			}

			err := utils.Validate.Struct(req)
			Expect(err).To(HaveOccurred())

			validationErr := errors.ValidatorError[TestValidationRequest](err)
			Expect(validationErr).NotTo(BeNil())

			// This simulates what happens in HandleError
			errorCode, errorMessage, httpCode, fields := errors.HandleAPIError(validationErr)

			// Verify the error info
			Expect(errorCode).To(Equal("ValidationError"))
			Expect(httpCode).To(Equal(400))
			Expect(errorMessage).To(ContainSubstring("Validation failed"))

			// Most importantly: fields should be populated!
			Expect(fields).NotTo(BeNil(), "Fields should not be nil")
			Expect(len(fields)).To(BeNumerically(">", 1), "Should have multiple validation errors")
			Expect(fields).To(HaveKey("id"))
			Expect(fields).To(HaveKey("name"))
			Expect(fields).To(HaveKey("email"))
		})
	})

	Describe("Single Validation Error", func() {
		It("should return a clear error message for a single validation failure", func() {
			req := TestValidationRequest{
				ID:       "test-id",
				Name:     "John",
				Password: "validpassword123", // Valid password (>= 8 chars)
				Email:    "invalid-email",     // Invalid email format
			}

			err := utils.Validate.Struct(req)
			Expect(err).To(HaveOccurred())

			validationErr := errors.ValidatorError[TestValidationRequest](err)
			Expect(validationErr).NotTo(BeNil())

			// Should have a clear error message
			errMsg := validationErr.Error()
			Expect(errMsg).To(ContainSubstring("email"))
		})
	})
})
