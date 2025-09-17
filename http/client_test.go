package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/kod2ulz/gostart/contracts"
	"github.com/kod2ulz/gostart/errors"
	gostartHttp "github.com/kod2ulz/gostart/http"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

// MockSession is a mock for the http.Session interface
type MockSession struct {
	Token string
}

func (m *MockSession) Authorization() string {
	return m.Token
}

var _ = Describe("Http Client", func() {

	type TestData struct {
		Message string `json:"message"`
	}

	var (
		server *httptest.Server
		ctx    context.Context
		logger *logrus.Entry
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = logrus.NewEntry(logrus.New())
		logger.Logger.SetOutput(GinkgoWriter)
	})

	AfterEach(func() {
		if server != nil {
			server.Close()
		}
	})

	Context("when making a successful GET request", func() {
		It("should return a successful response and parse data", func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				// The http client will unmarshal the whole response first, then the data field
				json.NewEncoder(w).Encode(contracts.DataResponse(TestData{Message: "Success"}))
			}))

			res := gostartHttp.Client[TestData](logger).BaseUrl(server.URL).Get(ctx, "/test")

			Expect(res.HasError()).To(BeFalse())
			Expect(res.Code()).To(Equal(http.StatusOK))

			var resultData TestData
			err := res.ParseDataTo(&resultData)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultData.Message).To(Equal("Success"))
		})
	})

	Context("when making a POST request", func() {
		It("should send the body and receive a response", func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body TestData
				Expect(json.NewDecoder(r.Body).Decode(&body)).To(Succeed())
				Expect(body.Message).To(Equal("Request Body"))
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(contracts.DataResponse(TestData{Message: "Created"}))
			}))

			res := gostartHttp.Client[TestData](logger).BaseUrl(server.URL).Body(TestData{Message: "Request Body"}).Post(ctx, "/test")

			Expect(res.HasError()).To(BeFalse())
			Expect(res.Code()).To(Equal(http.StatusCreated))

			var resultData TestData
			err := res.ParseDataTo(&resultData)
			Expect(err).ToNot(HaveOccurred())
			Expect(resultData.Message).To(Equal("Created"))
		})
	})

	Context("when the server returns an error", func() {
		It("should handle a structured error response", func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				// The client expects the full api.Response structure
				json.NewEncoder(w).Encode(contracts.ErrorResponse[TestData](errors.GeneralFailure[TestData](errors.Errorf("invalid input")).WithErrorCode("INVALID_INPUT")))
			}))

			res := gostartHttp.Client[TestData](logger).BaseUrl(server.URL).Get(ctx, "/bad-request")

			Expect(res.HasError()).To(BeTrue())
			Expect(res.Code()).To(Equal(http.StatusBadRequest))

			errorModel, ok := res.Error.(*errors.ErrorModel[TestData])
			Expect(ok).To(BeTrue())
			Expect(errorModel.Code).To(Equal("INVALID_INPUT"))
			Expect(errorModel.Message).To(Equal("invalid input"))
		})
	})
})
