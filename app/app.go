package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/logr"

	"github.com/gin-gonic/gin"
)

var (
	// global single instance of the _instance
	_instance *ap
)

type ap struct {
	router *gin.Engine
	log    *logr.Logger
	start  time.Time
	conf   *conf

	serviceId string

	osc    chan os.Signal
	ctx    context.Context
	cancel context.CancelFunc

	heartbeatHandlers bool
	handlers map[string]gin.HandlerFunc
}

type AppIniter func(*ap) error

func Init(opts ...AppIniter) *ap {
	if _instance != nil {
		return _instance
	}
	for i := range opts {
		opts[i](nil)
	}
	config.Load() // Load .env file

	if err := logr.Config(); err != nil {
		slog.Error("Application log initialisation failed", "error", err)
		panic(err)
	}
	logr.Log().Info("starting app initialisation")
	_instance = &ap{
		log:   logr.Log(),
		start: time.Now(),
		conf:  Conf(),
		osc:   make(chan os.Signal, 1),
	}
	_instance.ctx, _instance.cancel = context.WithCancel(context.Background())
	_instance.initAPI()
	return _instance
}

func (a *ap) Logger() *logr.Logger {
	return a.log
}

func (a *ap) Log() *logr.Logger {
	return a.log
}

func (a *ap) Context() context.Context {
	return a.ctx
}

func (a *ap) Ctx() context.Context {
	return a.ctx
}

func (a *ap) Router() *gin.Engine {
	return a.router
}

func (a *ap) R() *gin.Engine {
	return a.router
}

func (a *ap) Config() *conf {
	return a.conf
}

func Log() *logr.Logger {
	return instance().log
}

func (a *ap) Run(name ...string) {
	fmt.Println()
	a.tryRegisterConsul(name...)
	signal.Notify(a.osc, os.Interrupt, syscall.SIGTERM)
	startupMsg := "started"
	if a.router != nil {
		startupMsg += " with http router " + a.conf.Address()
		go a.router.Run(a.conf.Address())
	}
	a.log.Info(startupMsg)
	<-a.osc
	a.cancel()
	fmt.Println()
	a.shutdown()
	if err := config.Consul.Deregister(a.serviceId); err != nil {
		a.log.Warn("failed to deregister service from consul", "error", err)
	} else if a.serviceId != "" {
		a.log.Info("service successfully deregistered from consul")
	}
	a.log.Info("shutdown complete")
}

func (a *ap) tryRegisterConsul(name ...string) {
	id, err := config.Consul.RegisterFromEnv(name...)
	if err != nil {
		// Check if the error is because the endpoint is not configured.
		// In that case, it's a warning, not a fatal error.
		if strings.Contains(err.Error(), "consul address not configured") {
			a.log.Warn("skipping consul registration", "reason", "consul address not configured")
		} else {
			a.log.Error("service registration failed", "error", err)
		}
		return
	}
	a.serviceId = id
	a.log.Info("service successfully registered with consul", "id", a.serviceId)
}

func (a *ap) shutdown() {
	a.log.Info("shutting down")
}

func WithHandlerOverride(key string, handler gin.HandlerFunc) AppIniter {
	return func(a *ap) error {
		a.handlers[key] = handler
		return nil
	}
}

func WithStaticFileHandler(webPath, filePath string, ) AppIniter {
	return func(a *ap) error {
		if a == nil {
			return nil
		}
		a.router.StaticFile(webPath, filePath)
		return nil
	}
}

func WithHeartbeatHandlers() AppIniter {
	return func(a *ap) error {
		if a == nil {
			return nil
		}
		a.heartbeatHandlers = true
		return nil
	}
}

func (a *ap) initAPI(opts ...AppIniter) {
	a.handlers = map[string]gin.HandlerFunc{
		"ok": func(c *gin.Context) {
			c.JSON(http.StatusOK, "OK")
		},
		"stats": func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]interface{}{
				"host": a.conf.Host, "started": a.start, "app": a.conf.Name,
				"uptime": time.Since(a.start).Round(100 * time.Millisecond).String(),
			})
		},
	}

	a.router = gin.New()
	a.router.Use(api.JSONLogMiddleware(a.log), gin.Recovery(), cors.New(cors.Config{
		AllowOrigins:     a.conf.Http.AllowOrigins,
		AllowMethods:     a.conf.Http.AllowMethods,
		AllowHeaders:     a.conf.Http.AllowHeaders,
		ExposeHeaders:    a.conf.Http.ExposeHeaders,
		AllowCredentials: a.conf.Http.AllowCredentials,
		MaxAge: a.conf.Http.MaxAge,
	}))

	for i := range opts {
		opts[i](a)
	}

	if a.heartbeatHandlers {
		a.router.GET("/", a.handlers["ok"])
		a.router.GET("/ok", a.handlers["ok"])
		a.router.GET("/stats", a.handlers["stats"])
	}
}

func instance() *ap {
	if _instance == nil {
		panic("app instance not initialised")
	}
	return _instance
}