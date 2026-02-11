package api_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/contracts"
	gin_framework "github.com/kod2ulz/gostart/api/frameworks/gin"
)

var _ = Describe("Router Initialization", func() {
	var (
		router api.Router
		err    error
	)

	BeforeEach(func() {
		// Set Gin to test mode for all tests
		gin.SetMode(gin.TestMode)
	})

	AfterEach(func() {
		// Clean up any created routers
		router = nil
		err = nil
	})

	Describe("Default Router Configuration", func() {
		It("should create a router with default configuration", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())
			Expect(router).NotTo(BeNil())
		})

		It("should have underlying Gin engine accessible", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			underlying := router.Underlying()
			Expect(underlying).NotTo(BeNil())

			// Should be a Gin engine
			ginEngine, ok := underlying.(*gin.Engine)
			Expect(ok).To(BeTrue())
			Expect(ginEngine).NotTo(BeNil())
		})

		It("should have OpenAPI routes configured", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			// Test OpenAPI JSON endpoint
			req, _ := http.NewRequest("GET", "/openapi.json", nil)
			w := httptest.NewRecorder()

			// Need to get the underlying Gin engine to test
			ginEngine := router.Underlying().(*gin.Engine)
			ginEngine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
		})

		It("should have Swagger UI routes configured", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			// Test Swagger UI endpoint
			req, _ := http.NewRequest("GET", "/swagger", nil)
			w := httptest.NewRecorder()

			ginEngine := router.Underlying().(*gin.Engine)
			ginEngine.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("Router Configuration Options", func() {
		It("should enable recovery middleware by default", func() {
			config := api.DefaultRouterConfig()
			Expect(config.EnableRecovery).To(BeTrue())
		})

		It("should enable logging middleware by default", func() {
			config := api.DefaultRouterConfig()
			Expect(config.EnableLogging).To(BeTrue())
		})

		It("should allow custom middleware configuration", func() {
			config := api.DefaultRouterConfig()
			config.CustomMiddleware = []api.MiddlewareFunc{
				func(ctx contracts.RequestContext) (bool, error) {
					return true, nil
				},
			}

			router, err = gin_framework.NewRouter(config)
			Expect(err).To(BeNil())
			Expect(router).NotTo(BeNil())
		})

		It("should support static file configuration", func() {
			config := api.DefaultRouterConfig()
			config.StaticPaths = map[string]string{
				"/static": "./static",
			}

			router, err = gin_framework.NewRouter(config)
			Expect(err).To(BeNil())
			Expect(router).NotTo(BeNil())
		})

		It("should allow CORS configuration", func() {
			config := api.DefaultRouterConfig()
			config.AllowOrigins = []string{"https://example.com"}
			config.AllowMethods = []string{"GET", "POST"}

			router, err = gin_framework.NewRouter(config)
			Expect(err).To(BeNil())
			Expect(router).NotTo(BeNil())
		})
	})

	Describe("Router Interface Compliance", func() {
		It("should implement all HTTP methods", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			// Test that all HTTP methods are available and return router for chaining
			var testHandler api.HandlerFunc = func(ctx contracts.RequestContext) {}

			Expect(router.GET("/test", testHandler)).To(Equal(router))
			Expect(router.POST("/test", testHandler)).To(Equal(router))
			Expect(router.PUT("/test", testHandler)).To(Equal(router))
			Expect(router.DELETE("/test", testHandler)).To(Equal(router))
			Expect(router.PATCH("/test", testHandler)).To(Equal(router))
			Expect(router.OPTIONS("/test", testHandler)).To(Equal(router))
			Expect(router.HEAD("/test", testHandler)).To(Equal(router))
		})

		It("should support route grouping", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			groupCalled := false
			var testHandler api.HandlerFunc = func(ctx contracts.RequestContext) {}

			router.Group("/api", func(r api.Router) {
				groupCalled = true
				r.GET("/users", testHandler)
				r.POST("/users", testHandler)
			})

			Expect(groupCalled).To(BeTrue())
		})

		It("should support middleware chaining", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			var middleware api.MiddlewareFunc = func(ctx contracts.RequestContext) (bool, error) {
				return true, nil
			}

			Expect(router.Use(middleware)).To(Equal(router))
		})

		It("should support static file serving", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			Expect(router.StaticFile("/favicon.ico", "./favicon.ico")).To(Equal(router))
			Expect(router.Static("/assets", "./assets")).To(Equal(router))
		})
	})

	Describe("OpenAPI Integration", func() {
		It("should implement OpenAPIRouter interface", func() {
			config := api.DefaultRouterConfig()
			router, err = gin_framework.NewRouter(config)

			Expect(err).To(BeNil())

			// Check if router implements OpenAPIRouter interface
			if openAPIRouter, ok := router.(api.OpenAPIRouter); ok {
				Expect(openAPIRouter.GetOpenAPIHandler()).NotTo(BeNil())
				Expect(openAPIRouter.GetSwaggerUIHandler()).NotTo(BeNil())

				doc, err := openAPIRouter.GenerateOpenAPIDoc()
				Expect(err).To(BeNil())
				Expect(doc).NotTo(BeNil())
			} else {
				Fail("Router should implement OpenAPIRouter interface")
			}
		})
	})
})