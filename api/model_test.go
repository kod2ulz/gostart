package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/utils"
)

var _ = Describe("RequestModel", func() {

	type BookResponse struct {
		Name   string `json:"name"`
		Author string `json:"author"`
		Pages  int    `json:"pages"`
	}

	type CreateBookRequest struct {
		Name   string `json:"name"   validate:"required"`
		Author string `json:"author" validate:"required"`
		Pages  int    `json:"pages"  validate:"required,gt=200"`
		api.RequestModal[CreateBookRequest]
	}

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

			It("cannot alter value of original object", func() {
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
		var createBookHandler = func(ctx context.Context) (out BookResponse, err api.Error) {
			param := CreateBookRequest{}
			if er := param.LoadFromContext(ctx, &param); err != nil {
				err = api.RequestLoadError[CreateBookRequest](er)
			} else {
				out = BookResponse{Name: param.Name, Author: param.Author, Pages: param.Pages}
			}
			return
		}

		var setupRouter = func() *gin.Engine {
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.POST("/books", api.HandlerWithResponse[CreateBookRequest](createBookHandler))
			return router
		}

		BeforeEach(func() {
			router = setupRouter()
			recorder = httptest.NewRecorder()
		})

		Context("with post data", func() {

			It("can load request model from gin router request", func() {
				var res *ResultModel[CreateBookRequest, BookResponse]
				payload := jsonDataOf("name", "Book 1", "author", "TestBot1", "pages", 400)
				router.ServeHTTP(recorder, makeRequest(http.MethodPost, "/books", payload))
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
				Expect(res).ToNot(BeNil())
				Expect(res.HasError()).To(BeFalse())
				var param BookResponse
				Expect(utils.StructCopy(res.Data(), &param)).To(BeNil())
				Expect(param).To(Equal(BookResponse{Name: "Book 1", Author: "TestBot1", Pages: 400}))
			})

			It("can validate request and fail on invalid parameters", func() {
				var res *ResultModel[CreateBookRequest, any]
				payload := jsonDataOf("author", "TestBot2", "pages", 50)
				router.ServeHTTP(recorder, makeRequest(http.MethodPost, "/books", payload))
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
