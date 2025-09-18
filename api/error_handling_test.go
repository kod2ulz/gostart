package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
	gin_framework "github.com/kod2ulz/gostart/api/frameworks/gin"
	"github.com/kod2ulz/gostart/utils"
)

var _ = Describe("Error Handling Integration Tests", func() {
	var (
		router   *gin.Engine
		recorder *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		router = gin.New()
		recorder = httptest.NewRecorder()
	})

	AfterEach(func() {
		router = nil
		recorder = nil
	})

	Describe("Request Loading Errors", func() {
		It("should handle invalid JSON body", func() {
			type TestRequest struct {
				Name string `json:"name" validate:"required"`
				api.RequestModal[TestRequest]
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req TestRequest
				var modal api.RequestModal[TestRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[TestRequest](loadError)
				} else {
					req = loaded.(TestRequest)
				}

				return "Hello, " + req.Name, nil
			})

			router.POST("/test", gin_framework.WrapHandler(handler))

			// Send invalid JSON
			req, _ := http.NewRequest("POST", "/test", strings.NewReader("{invalid json"))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})

		It("should handle missing required fields", func() {
			type TestRequest struct {
				Name string `json:"name" validate:"required"`
				Age  int    `json:"age" validate:"required"`
				api.RequestModal[TestRequest]
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req TestRequest
				var modal api.RequestModal[TestRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[TestRequest](loadError)
				} else {
					req = loaded.(TestRequest)
				}

				// Explicitly validate the struct - this is needed because RequestModal doesn't auto-validate
				if validateErr := utils.Validate.Struct(req); validateErr != nil {
					return "", errors.ValidatorError[TestRequest](validateErr)
				}

				return "Hello, " + req.Name, nil
			})

			router.POST("/required", gin_framework.WrapHandler(handler))

			// Send JSON with missing required field
			req, _ := http.NewRequest("POST", "/required", strings.NewReader(`{"name": "John"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})
	})

	Describe("Validation Errors", func() {
		It("should handle email validation", func() {
			type UserRequest struct {
				Email string `json:"email" validate:"required,email"`
				api.RequestModal[UserRequest]
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req UserRequest
				var modal api.RequestModal[UserRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[UserRequest](loadError)
				} else {
					req = loaded.(UserRequest)
				}

				// Explicitly validate the struct
				if validateErr := utils.Validate.Struct(req); validateErr != nil {
					return "", errors.ValidatorError[UserRequest](validateErr)
				}

				return "Email: " + req.Email, nil
			})

			router.POST("/email", gin_framework.WrapHandler(handler))

			// Test with invalid email
			req, _ := http.NewRequest("POST", "/email", strings.NewReader(`{"email": "invalid-email"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})

		It("should handle numeric range validation", func() {
			type RangeRequest struct {
				Age int `json:"age" validate:"gte=18,lte=120"`
				api.RequestModal[RangeRequest]
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req RangeRequest
				var modal api.RequestModal[RangeRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[RangeRequest](loadError)
				} else {
					req = loaded.(RangeRequest)
				}

				// Explicitly validate the struct
				if validateErr := utils.Validate.Struct(req); validateErr != nil {
					return "", errors.ValidatorError[RangeRequest](validateErr)
				}

				return fmt.Sprintf("Age: %d", req.Age), nil
			})

			router.POST("/range", gin_framework.WrapHandler(handler))

			// Test with age below minimum
			req, _ := http.NewRequest("POST", "/range", strings.NewReader(`{"age": 15}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})
	})

	Describe("Service Layer Errors", func() {
		It("should handle not found errors", func() {
			type FindRequest struct {
				ID string `json:"id" validate:"required"`
				api.RequestModal[FindRequest]
			}

			// Simulate a service that might not find an item
			findUser := func(id string) (string, ierrors.Error) {
				if id == "existing-user" {
					return "John Doe", nil
				}
				return "", errors.NotFound[string, string]("user not found")
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req FindRequest
				var modal api.RequestModal[FindRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[FindRequest](loadError)
				} else {
					req = loaded.(FindRequest)
				}

				return findUser(req.ID)
			})

			router.POST("/find", gin_framework.WrapHandler(handler))

			// Test with non-existent user
			req, _ := http.NewRequest("POST", "/find", strings.NewReader(`{"id": "non-existent"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusNotFound))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})

		It("should handle conflict errors", func() {
			type CreateRequest struct {
				Email string `json:"email" validate:"required,email"`
				api.RequestModal[CreateRequest]
			}

			// Simulate a service that checks for duplicates
			createUser := func(email string) (string, ierrors.Error) {
				if email == "existing@example.com" {
					return "", errors.GeneralFailure[CreateRequest](errors.Errorf("email already exists")).WithErrorCodeAndHttpStatusCode("CONFLICT", http.StatusConflict)
				}
				return "User created", nil
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req CreateRequest
				var modal api.RequestModal[CreateRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[CreateRequest](loadError)
				} else {
					req = loaded.(CreateRequest)
				}

				return createUser(req.Email)
			})

			router.POST("/create", gin_framework.WrapHandler(handler))

			// Test with duplicate email
			req, _ := http.NewRequest("POST", "/create", strings.NewReader(`{"email": "existing@example.com"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusConflict))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})
	})

	Describe("Authorization and Permission Errors", func() {
		It("should handle unauthorized access", func() {
			type SecureRequest struct {
				Action string `json:"action" validate:"required"`
				api.RequestModal[SecureRequest]
			}

			// Simulate an authorization check
			checkPermission := func(action string) (string, ierrors.Error) {
				if action == "admin-only" {
					return "", errors.ServiceUnauthorised(errors.Errorf("insufficient permissions"))
				}
				return "Action performed", nil
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req SecureRequest
				var modal api.RequestModal[SecureRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[SecureRequest](loadError)
				} else {
					req = loaded.(SecureRequest)
				}

				return checkPermission(req.Action)
			})

			router.POST("/secure", gin_framework.WrapHandler(handler))

			// Test with restricted action
			req, _ := http.NewRequest("POST", "/secure", strings.NewReader(`{"action": "admin-only"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})
	})

	Describe("Internal Server Errors", func() {
		It("should handle unexpected errors gracefully", func() {
			type RiskyRequest struct {
				Operation string `json:"operation" validate:"required"`
				api.RequestModal[RiskyRequest]
			}

			// Simulate an operation that might fail unexpectedly
			riskyOperation := func(operation string) (string, ierrors.Error) {
				if operation == "fail" {
					return "", errors.GeneralFailure[RiskyRequest](errors.Errorf("database connection failed"))
				}
				return "Operation successful", nil
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req RiskyRequest
				var modal api.RequestModal[RiskyRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[RiskyRequest](loadError)
				} else {
					req = loaded.(RiskyRequest)
				}

				return riskyOperation(req.Operation)
			})

			router.POST("/risky", gin_framework.WrapHandler(handler))

			// Test with failing operation
			req, _ := http.NewRequest("POST", "/risky", strings.NewReader(`{"operation": "fail"}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())
			Expect(response["success"]).To(Equal(false))
			Expect(response["error"]).NotTo(BeNil())
		})
	})

	Describe("Error Response Structure", func() {
		It("should maintain consistent error response format", func() {
			type TestRequest struct {
				Value string `json:"value" validate:"required"`
				api.RequestModal[TestRequest]
			}

			handler := api.JSONHandler[string](func(ctx contracts.RequestContext) (string, ierrors.Error) {
				var req TestRequest
				var modal api.RequestModal[TestRequest]

				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return "", errors.RequestLoadFailed[TestRequest](loadError)
				} else {
					req = loaded.(TestRequest)
				}

				// Explicitly validate the struct
				if validateErr := utils.Validate.Struct(req); validateErr != nil {
					return "", errors.ValidatorError[TestRequest](validateErr)
				}

				return "Value: " + req.Value, nil
			})

			router.POST("/consistent", gin_framework.WrapHandler(handler))

			// Test with missing required field
			req, _ := http.NewRequest("POST", "/consistent", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(recorder, req)

			Expect(recorder.Code).To(Equal(http.StatusBadRequest))

			var response map[string]interface{}
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).To(BeNil())

			// Check standard error response structure
			Expect(response).To(HaveKey("success"))
			Expect(response["success"]).To(Equal(false))
			Expect(response).To(HaveKey("error"))
			Expect(response["error"]).NotTo(BeNil())
			Expect(response).To(HaveKey("time"))
			Expect(response["time"]).NotTo(BeNil())
		})
	})
})