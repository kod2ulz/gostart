package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	gin_framework "github.com/kod2ulz/gostart/api/frameworks/gin"
	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	"github.com/kod2ulz/gostart/ierrors"
)

var _ = Describe("API Handler Patterns", func() {

	var router *gin.Engine
	var recorder *httptest.ResponseRecorder
	var testUserService *UserService

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		router = gin.New()
		testUserService = NewUserService()
		recorder = httptest.NewRecorder()
	})

	When("using the new Handler patterns with RequestContext", func() {
		It("should handle single entity creation using JSONHandler[T]", func() {
			// Setup route using the new JSONHandler pattern
			router.POST("/users", gin_framework.WrapHandler(api.JSONHandler[User](func(ctx contracts.RequestContext) (User, ierrors.Error) {
				// Load request from HTTP
				var req CreateUserRequest
				var modal api.RequestModal[CreateUserRequest]
				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return User{}, errors.RequestLoadFailed[CreateUserRequest](loadError)
				} else {
					req = loaded.(CreateUserRequest)
				}

				// Business logic for creating user
				return testUserService.CreateUser(&req)
			})))

			// Test successful creation
			userReq := CreateUserRequest{
				Name:  "John Doe",
				Email: "john@example.com",
				Age:   25,
			}

			jsonValue, _ := json.Marshal(userReq)
			req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonValue))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response contracts.Response[User]
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response.Success).To(BeTrue())
			var user User
			Expect(response.ParseDataTo(&user)).To(BeNil())
			Expect(user.Name).To(Equal("John Doe"))
			Expect(user.Email).To(Equal("john@example.com"))
			Expect(user.Age).To(Equal(25))
		})

		It("should handle list response using JSONHandler[T] with pagination", func() {
			// Setup route using the JSONHandler pattern for lists
			router.GET("/users", gin_framework.WrapHandler(api.JSONHandler[[]User](func(ctx contracts.RequestContext) ([]User, ierrors.Error) {
				// Business logic for listing users - load request from context
				return testUserService.ListUsersDirect(ctx)
			})))

			// Seed some test data
			testUserService.seedUsers(5)

			req, _ := http.NewRequest("GET", "/users", nil)
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response contracts.Response[[]User]
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response.Success).To(BeTrue())
			var users []User
			Expect(response.ParseDataTo(&users)).To(BeNil())
			Expect(len(users)).To(BeNumerically(">", 0))
		})

		It("should handle single entity retrieval using JSONHandler[T]", func() {
			// First, create a user to retrieve
			createdUser, err := testUserService.CreateUser(&CreateUserRequest{
				Name:  "Jane Doe",
				Email: "jane@example.com",
				Age:   30,
			})
			Expect(err).To(BeNil())

			// Setup route using the new JSONHandler pattern
			router.GET("/users/:id", gin_framework.WrapHandler(api.JSONHandler[User](func(ctx contracts.RequestContext) (User, ierrors.Error) {
				// Extract ID from context (path parameter)
				id := ctx.Param("id").String()
				if id == "" {
					return User{}, errors.GeneralFailure[any](fmt.Errorf("missing user ID")).WithErrorCodeAndHttpStatusCode("BAD_REQUEST", http.StatusBadRequest)
				}

				// Business logic for getting user by ID
				return testUserService.GetUser(id)
			})))

			req, _ := http.NewRequest("GET", fmt.Sprintf("/users/%s", createdUser.ID), nil)
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response contracts.Response[User]
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response.Success).To(BeTrue())
			var user User
			Expect(response.ParseDataTo(&user)).To(BeNil())
			Expect(user.Name).To(Equal("Jane Doe"))
			Expect(user.Email).To(Equal("jane@example.com"))
		})

		It("should handle not found errors properly", func() {
			router.GET("/users/:id", gin_framework.WrapHandler(api.JSONHandler[User](func(ctx contracts.RequestContext) (User, ierrors.Error) {
				id := ctx.Param("id").String()
				return testUserService.GetUser(id)
			})))

			req, _ := http.NewRequest("GET", "/users/nonexistent-id", nil)
			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusNotFound))

			var response contracts.Response[User]
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response.Success).To(BeFalse())
		})

		It("should handle validation errors properly", func() {
			router.POST("/users", gin_framework.WrapHandler(api.JSONHandler[User](func(ctx contracts.RequestContext) (User, ierrors.Error) {
				// Load request from HTTP
				var req CreateUserRequest
				var modal api.RequestModal[CreateUserRequest]
				if loaded, loadError := modal.RequestLoad(ctx); loadError != nil {
					return User{}, errors.RequestLoadFailed[CreateUserRequest](loadError)
				} else {
					req = loaded.(CreateUserRequest)
				}

				// Business logic for creating user
				return testUserService.CreateUser(&req)
			})))

			// Test with invalid data (missing required fields)
			invalidReq := map[string]interface{}{
				"name":  "",              // Empty name should fail validation
				"email": "invalid-email", // Invalid email
				"age":   15,              // Age below minimum
			}

			jsonValue, _ := json.Marshal(invalidReq)
			req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(jsonValue))
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(recorder, req)
			Expect(recorder.Code).To(Equal(http.StatusBadRequest))

			var response contracts.Response[User]
			json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(response.Success).To(BeFalse())
		})
	})

	When("using the same service methods with MQ handlers", func() {
		It("should handle user creation via MQ", func() {
			// Test the same service method but called from MQ context
			mqHandler := func(ctx context.Context, msg interface{}) (interface{}, ierrors.Error) {
				// Simulate MQ message processing
				userReq, ok := msg.(*CreateUserRequest)
				if !ok {
					return nil, errors.GeneralFailure[CreateUserRequest](fmt.Errorf("invalid message type")).WithErrorCodeAndHttpStatusCode("BAD_REQUEST", http.StatusBadRequest)
				}

				// Use the same service method
				user, err := testUserService.CreateUser(userReq)
				if err != nil {
					return nil, err
				}

				return UserCreatedEvent{
					UserID:    user.ID,
					Name:      user.Name,
					Email:     user.Email,
					Timestamp: time.Now(),
				}, nil
			}

			// Test the handler
			userReq := &CreateUserRequest{
				Name:  "MQ User",
				Email: "mq@example.com",
				Age:   28,
			}

			result, err := mqHandler(context.Background(), userReq)
			Expect(err).To(BeNil())
			Expect(result).NotTo(BeNil())

			event, ok := result.(UserCreatedEvent)
			Expect(ok).To(BeTrue())
			Expect(event.Name).To(Equal("MQ User"))
			Expect(event.Email).To(Equal("mq@example.com"))
		})
	})
})

// Test types and service implementation for the new handler patterns

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Age       int       `json:"age"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateUserRequest struct {
	Name  string `json:"name" validate:"required" example:"John Doe"`
	Email string `json:"email" validate:"required,email" example:"john@example.com"`
	Age   int    `json:"age" validate:"required,gte=18" example:"25"`
	api.RequestModal[CreateUserRequest]
}

func (r CreateUserRequest) RequestLoad(ctx contracts.RequestContext) (param contracts.RequestParam, err error) {
	var out CreateUserRequest
	if loadErr := out.LoadFromJsonBody(ctx, &out); loadErr != nil {
		return param, errors.RequestLoadFailed[any](loadErr)
	}
	if ctxSetter, ok := ctx.(interface{ Set(string, interface{}) }); ok {
		ctxSetter.Set(out.ContextKey(), out)
	}
	return out, nil
}

// Use the standard api.ListRequest for pagination
type ListUsersRequest = api.ListRequest

type UserCreatedEvent struct {
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

type UserService struct {
	users map[string]User
}

func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]User),
	}
}

func (s *UserService) CreateUser(req *CreateUserRequest) (User, ierrors.Error) {
	// Validate request
	if req.Name == "" {
		return User{}, errors.ValidationFailed[CreateUserRequest](fmt.Errorf("name is required"))
	}
	if req.Age < 18 {
		return User{}, errors.ValidationFailed[CreateUserRequest](fmt.Errorf("age must be at least 18"))
	}

	// Check if email already exists
	for _, user := range s.users {
		if user.Email == req.Email {
			return User{}, errors.GeneralFailure[CreateUserRequest](fmt.Errorf("email already exists")).WithErrorCodeAndHttpStatusCode("CONFLICT", http.StatusConflict)
		}
	}

	// Create user
	user := User{
		ID:        uuid.New().String(),
		Name:      req.Name,
		Email:     req.Email,
		Age:       req.Age,
		CreatedAt: time.Now(),
	}

	s.users[user.ID] = user
	return user, nil
}

func (s *UserService) GetUser(id string) (User, ierrors.Error) {
	user, exists := s.users[id]
	if !exists {
		return User{}, errors.NotFound[string, string]("user not found")
	}
	return user, nil
}

func (s *UserService) ListUsers(ctx contracts.RequestContext, param contracts.RequestParam) ([]User, *int64, ierrors.Error) {
	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	// Extract pagination from param using interface checks
	var start, end int

	// Check if param has GetOffset method
	if listParam, ok := param.(interface{ GetOffset() int }); ok {
		start = listParam.GetOffset()
	} else {
		start = 0
	}

	// Check if param has GetLimit method
	if listParam, ok := param.(interface{ GetLimit() int }); ok {
		end = start + listParam.GetLimit()
	} else {
		end = len(users)
	}

	// Apply bounds
	if start > len(users) {
		start = len(users)
	}
	if end > len(users) {
		end = len(users)
	}

	paginatedUsers := users[start:end]
	total := int64(len(users))

	return paginatedUsers, &total, nil
}

// ListUsersDirect loads request from context directly (for JSONHandler pattern)
func (s *UserService) ListUsersDirect(ctx contracts.RequestContext) ([]User, ierrors.Error) {
	var param api.ListRequest

	// For GET requests, we need to load the request first using ListRequest's own RequestLoad method
	if loaded, loadError := param.RequestLoad(ctx); loadError != nil {
		return nil, errors.RequestLoadFailed[api.ListRequest](loadError)
	} else {
		param = loaded.(api.ListRequest)
	}

	users := make([]User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	// Apply pagination using the loaded param
	start := param.GetOffset()
	end := start + param.GetLimit()

	// Apply bounds
	if start > len(users) {
		start = len(users)
	}
	if end > len(users) {
		end = len(users)
	}

	return users[start:end], nil
}

func (s *UserService) seedUsers(count int) {
	for i := 0; i < count; i++ {
		user := User{
			ID:        uuid.New().String(),
			Name:      fmt.Sprintf("User %d", i+1),
			Email:     fmt.Sprintf("user%d@example.com", i+1),
			Age:       20 + i,
			CreatedAt: time.Now(),
		}
		s.users[user.ID] = user
	}
}
