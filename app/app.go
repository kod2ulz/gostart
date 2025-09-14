package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/config"
	"github.com/kod2ulz/gostart/logr"
	

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
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

func (a *ap) Register(name ...string) (err error) {
	// Ensure the shared consul client is configured
	if _, err = config.Consul.Endpoint(); err != nil {
		a.log.Warn("skipping consul registration", "error", err)
		return nil // Do not block app startup if consul is not available
	}

	client := config.Consul.Client()
	if client == nil {
		a.log.Warn("skipping consul registration: client not configured")
		return nil
	}

	var serviceName string
	var env = config.Env.Helper("CONSUL")
	var serviceHost = env.Get("SERVICE_HOST", a.conf.Host).String()

	if len(name) > 0 && name[0] != "" {
		serviceName = name[0]
	} else {
		serviceName = env.Get("SERVICE_NAME", a.conf.Name).String()
	}
	if id := env.Get("SERVICE_ID"); id.Valid() {
		a.serviceId = id.String()
	} else if autoId := env.Get("SERVICE_ID_AUTO"); autoId.Bool() {
		a.serviceId = fmt.Sprintf("%s-%s-%s", serviceName, serviceHost, a.conf.Version)
	} else {
		a.serviceId = serviceName
	}

	if err = client.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID:      a.serviceId,
		Name:    serviceName,
		Port:    a.conf.HttpPort,
		Address: serviceHost,
		Tags:    []string{a.conf.Version, serviceName, serviceHost, a.start.In(a.conf.Location).Format(time.RFC1123Z)},
		Check: &consulapi.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%v/ok", a.conf.Host, a.conf.HttpPort),
			Interval: a.conf.Uptime.Interval.String(),
			Timeout:  a.conf.Uptime.Timeout.String(),
		},
	}); err != nil {
		a.log.Error("service registration failed", "error", err)
		return err
	}
	a.log.Info("service successfully registered with consul")
	return nil
}

func (a *ap) Run() {
	fmt.Println()
	if err := a.Register(); err != nil {
		a.log.Error("failed to register service with consul", "error", err)
	}
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
	if client := config.Consul.Client(); client != nil && a.serviceId != "" {
		client.Agent().ServiceDeregister(a.serviceId)
	}
	a.log.Info("shutdown complete")
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