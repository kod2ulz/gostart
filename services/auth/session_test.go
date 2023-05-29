package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/services/auth"
	"github.com/kod2ulz/gostart/utils"
)

var (
	testTimeout = 2 * time.Second
)

var _ = Describe("Session Service", func() {

	Context("an unregistered user", func() {
		var signupError error
		var user auth.UserData
		var ctx context.Context
		var cancel context.CancelFunc
		var userStore auth.SessionStore[uuid.UUID, auth.UserData]
		var sessionService *auth.GenericSessionService[uuid.UUID, auth.UserData]
		var signupReq = auth.SignupRequest{Username: "user@test.com", Password: "123@Paswerd"}

		AfterEach(func() { userStore.Clear() })

		BeforeEach(func() {
			userStore = auth.InMemoryUserStore()
			sessionService = auth.SessionService(log, userStore)
			ctx, cancel = context.WithTimeout(context.Background(), testTimeout)
			user, signupError = sessionService.Signup(signupReq.InContext(ctx, signupReq))
		})

		When("Doing Registration", func() {

			It("should successfully sign up a user", func() {
				defer cancel()
				Expect(signupError).To(BeNil())
				Expect(user.Email).To(Equal(signupReq.Username))
				Expect(user.Password).ToNot(BeEmpty())
				Expect(user.Password).ToNot(Equal(signupReq.Password))
			})

		})
	})

	Context("a regitered user", func() {

		When("not logged in", func() {
			var router *gin.Engine
			var recorder *httptest.ResponseRecorder

			userStore := auth.InMemoryUserStore()
			sessionService := auth.SessionService(log, userStore)
			var signupReq auth.SignupRequest

			AfterEach(func() { userStore.Clear() })

			BeforeEach(func(ctx context.Context) {
				signupReq = createSignupRequest()
				registerUser(ctx, signupReq, sessionService)
				router = utils.Test.GinRouter(func(e *gin.Engine) {
					sessionService.API(e.Group("/auth"))
				})
				recorder = httptest.NewRecorder()
			})

			It("should be able to login and get a session token", func(ctx context.Context) {
				_, cancel := context.WithTimeout(ctx, testTimeout)
				defer cancel()
				var token auth.TokenResponse
				var res api.Response[auth.TokenResponse]
				payload := utils.Test.JsonEncode(signupReq)
				router.ServeHTTP(recorder, utils.Test.Request(http.MethodPost, "/auth/login", payload))
				checkResponse(recorder, &res)
				Expect(res.Data).ToNot(BeNil())
				utils.StructCopy(res.Data, &token)
				Expect(token.TokenType).To(Equal(auth.TokenTypeBearer))
				Expect(token.AccessToken).ToNot(BeEmpty())
				Expect(token.RefreshToken).ToNot(BeEmpty())
				Expect(token.ExpiresIn > 0).To(BeTrue())
			})

		})

		When("logged in", func() {
			var router *gin.Engine
			var token auth.TokenResponse
			var recorder *httptest.ResponseRecorder
			var signupReq auth.SignupRequest

			userStore := auth.InMemoryUserStore()
			sessionService := auth.SessionService(log, userStore)

			router = utils.Test.GinRouter(func(e *gin.Engine) {
				sessionService.API(e.Group("/auth"))
			})

			BeforeEach(func(ctx context.Context) {
				signupReq = createSignupRequest()
				registerUser(ctx, signupReq, sessionService)
				token = authenticateUser(ctx, signupReq, sessionService)
				recorder = httptest.NewRecorder()
			})

			AfterEach(func() { userStore.Clear() })

			It("should be able to verify an issued token", func(ctx context.Context) {
				_, cancel := context.WithTimeout(ctx, testTimeout)
				defer cancel()
				var user auth.User
				var res api.Response[api.User]
				payload := utils.Test.JsonDataOf("token", token.AccessToken)
				router.ServeHTTP(recorder, utils.Test.Request(http.MethodPost, "/auth/verify", payload))
				checkResponse(recorder, &res)
				Expect(res.Data).ToNot(BeNil())
				utils.StructCopy(res.Data, &user)
				Expect(user.Email).To(Equal(signupReq.Username))
				Expect(user.Claims).ToNot(BeNil())
				Expect(user.Claims.ExpiresAt.After(time.Now())).To(BeTrue())
			})

			It("should be able to refresh a token", func(ctx context.Context) {
				_, cancel := context.WithTimeout(ctx, testTimeout)
				defer cancel()
				var newToken auth.TokenResponse
				var res api.Response[auth.TokenResponse]
				payload := utils.Test.JsonDataOf("refresh_token", token.RefreshToken)
				router.ServeHTTP(recorder, utils.Test.Request(http.MethodPost, "/auth/refresh", payload))
				checkResponse(recorder, &res)
				Expect(res.Data).ToNot(BeNil())
				utils.StructCopy(res.Data, &newToken)
				Expect(newToken.TokenType).To(Equal(auth.TokenTypeBearer))
				Expect(newToken.AccessToken).ToNot(BeEmpty())
				Expect(newToken.RefreshToken).ToNot(BeEmpty())
				Expect(newToken.ExpiresIn > 0).To(BeTrue())
			})

		})

	})

})
