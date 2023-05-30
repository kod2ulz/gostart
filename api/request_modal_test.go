package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/utils"
)

var _ = Describe("RequestModal", func() {

	Describe("Context Behaviour", func() {

		Context("by default", func() {
			It("will have a context key matching pointer type of request model", func() {
				var param = CreateBookRequest{}
				Expect(param.ContextKey()).To(Equal(fmt.Sprintf("%T", &param)))
			})
		})

		Context("when loaded into context", func() {
			var book = CreateBookRequest{Name: "ABC", Author: "Alphabet"}
			var ctx = context.WithValue(context.TODO(), book.ContextKey(), book)

			It("can be read back out of the context", func() {
				var param = CreateBookRequest{}
				Expect(param.LoadFromContext(ctx, &param)).To(BeNil())
				Expect(param).To(Equal(book))
			})

			It("cannot change value of original object", func() {
				var param = CreateBookRequest{}
				Expect(param.LoadFromContext(ctx, &param)).To(BeNil())
				param.Pages = 200
				Expect(param.Pages).ToNot(Equal(book.Pages))
			})
		})

		Context("when passed into parent context", func() {
			var book = CreateBookRequest{Name: "ABC", Author: "Alphabet"}

			It("can write itself to another context", func() {
				var ctx = book.InContext(context.TODO(), book)
				Expect(ctx.Value(book.ContextKey())).ToNot(BeNil())
				Expect(ctx.Value(book.ContextKey())).To(Equal(book))
			})

			It("can be extracted from context", func() {
				var ctx = book.InContext(context.TODO(), book)
				var param = CreateBookRequest{}
				Expect(param.ContextKey()).To(Equal(book.ContextKey()))
				Expect(param.LoadFromContext(ctx, &param)).To(BeNil())
				Expect(param).To(Equal(book))
			})
		})

	})

	Describe("Loading from HTTP Request", func() {

		var router *gin.Engine
		var recorder *httptest.ResponseRecorder
		books := bookService()

		When("posting data with unauthenticated user", func() {

			BeforeEach(func() {
				router = utils.Test.GinRouter(func(e *gin.Engine) {
					books.setRoutes(e.Group("/books"))
				})
				recorder = httptest.NewRecorder()
			})
			AfterEach(func() { books.clear() })

			It("can load request model from gin router request", func() {
				var res *ResultModel[CreateBookRequest, Book]
				id := uuid.New()
				payload := utils.Test.JsonDataOf("id", id, "name", "Book 1", "author", "TestBot1", "pages", 400)
				router.ServeHTTP(recorder, utils.Test.Request(http.MethodPost, "/books", payload))
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
				Expect(res).ToNot(BeNil())
				Expect(res.HasError()).To(BeFalse())
				var param Book
				Expect(utils.StructCopy(res.Data(), &param)).To(BeNil())
				Expect(param).To(Equal(Book{ID: id, Name: "Book 1", Author: "TestBot1", Pages: 400}))
			})

			It("can validate request and fail on invalid parameters", func() {
				var res *ResultModel[CreateBookRequest, any]
				payload := utils.Test.JsonDataOf("author", "TestBot2", "pages", 50)
				router.ServeHTTP(recorder, utils.Test.Request(http.MethodPost, "/books", payload))
				Expect(recorder.Code).To(Equal(http.StatusBadRequest))
				Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
				Expect(res).ToNot(BeNil())
				Expect(res.Error().Message).ToNot(BeEmpty())
				Expect(res.Error().Code).To(Equal(api.ErrorCodeValidatorError))
				Expect(len(res.Error().Fields)).To(Equal(2))
			})
		})

		
	})

})
