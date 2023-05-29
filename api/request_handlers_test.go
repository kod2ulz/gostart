package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/services/auth"
	"github.com/kod2ulz/gostart/utils"
)

var _ = Describe("Request Handler", func() {

	var router *gin.Engine
	var recorder *httptest.ResponseRecorder
	books := bookService()

	When("posting data with authenticated user", func() {
		var user auth.UserData
		headers := api.Headers{}
		userStore := auth.InMemoryUserStore()
		sessionService := auth.SessionService(nil, userStore)
		router = utils.Test.GinRouter(func(e *gin.Engine) {
			books.setRoutes(e.Group("/books"), sessionService.Auther())
		})

		BeforeEach(func(ctx context.Context) {
			signupReq := createSignupRequest()
			user = registerUser(ctx, signupReq, sessionService)
			token := authenticateUser(ctx, signupReq, sessionService)
			headers.WithBearerToken(token.AccessToken)
			recorder = httptest.NewRecorder()
		})

		AfterEach(func() { books.clear() })

		It("can load post data from request using custom defined request loader", func() {
			var res api.Response[Book]
			payload := utils.Test.JsonDataOf("name", "Book 5", "author", "CreateBot2", "pages", 600)
			router.ServeHTTP(recorder, utils.Test.Request(http.MethodPost, "/books", payload, headers))
			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
			Expect(res.Success).To(BeTrue())
			Expect(res.Error).To(BeNil())
			var book Book
			Expect(res.ParseDataTo(&book)).To(BeNil())
			Expect(book.CreatedBy).ToNot(BeNil())
			Expect(*book.CreatedBy).To(Equal(user.GetID()))
		})

		It("can read back list data formatted as Response[T]", func() {
			var res api.Response[[]Book]
			_, createErr := books.seed(45, &user)
			Expect(createErr).To(BeNil())
			router.ServeHTTP(recorder, utils.Test.Request(http.MethodGet, "/books", nil, headers))
			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
			Expect(res.Success).To(BeTrue())
			Expect(res.Error).To(BeNil())
			Expect(res.Meta).ToNot(BeNil())
			var book []Book
			Expect(res.ParseDataTo(&book)).To(BeNil())
			Expect(res.Meta.Limit).To(BeEquivalentTo(len(book)))
			Expect(res.Meta.Total).To(BeEquivalentTo(45))
		})

		It("can get back single entity response by ID", func() {
			newBooks, _ := books.seed(1, &user)
			Expect(newBooks).ToNot(BeEmpty())
			bookID, createdBook := newBooks[0].ID.String(), *newBooks[0]
			var res api.Response[Book] //= api.EmptyResponse[Book]()
			router.ServeHTTP(recorder, utils.Test.Request(http.MethodGet, "/books/"+bookID, []byte{}, headers))
			Expect(recorder.Code).To(Equal(http.StatusOK))
			Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
			Expect(res.Success).To(BeTrue())
			Expect(res.Error).To(BeNil())
			Expect(res.Meta).To(BeNil())
			var book Book
			Expect(res.ParseDataTo(&book)).To(BeNil())
			Expect(book).To(Equal(createdBook))
		})
	})
})
