package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/utils"
)

var _ = Describe("Request Handler", func() {

	Describe("Loading from HTTP Request", func() {

		var router *gin.Engine
		var recorder *httptest.ResponseRecorder
		books := bookService()
		router = utils.Test.GinRouter(func(e *gin.Engine) {
			books.setRoutes(e.Group("/books"))
		})

		BeforeEach(func() { recorder = httptest.NewRecorder() })
		AfterEach(func() { books.clear() })

		Context("with custom handler", func() {

			It("can post data and return api.Response", func() {
				var res api.Response[Book]
				id := uuid.New()
				payload := jsonDataOf("id", id, "name", "Book 1", "author", "TestBot1", "pages", 400)
				router.ServeHTTP(recorder, makeRequest(http.MethodPost, "/books", payload))
				Expect(recorder.Code).To(Equal(http.StatusOK))
				Expect(json.NewDecoder(recorder.Body).Decode(&res)).To(BeNil())
				Expect(res.Success).To(BeTrue())
				Expect(res.Error).To(BeNil())
				Expect(res.Meta).To(BeNil())
				Expect((time.Now().Unix() - res.Timestamp) < 10).To(BeTrue())
				var data Book
				Expect(utils.StructCopy(res.Data, &data)).To(BeNil())
				Expect(data).To(Equal(Book{ID: id, Name: "Book 1", Author: "TestBot1", Pages: 400}))
			})

		})
	})
})
