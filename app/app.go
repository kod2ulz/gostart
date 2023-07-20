package app

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/kod2ulz/gostart/api"
	"github.com/kod2ulz/gostart/logr"
	"github.com/kod2ulz/gostart/utils"
	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/joho/godotenv"
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
	consul *consulapi.Client

	osc    chan os.Signal
	ctx    context.Context
	cancel context.CancelFunc
}

func Init(strict ...bool) (*ap) {
	if _instance != nil {
		return _instance
	}
	var _strict bool
	if len(strict) > 0 {
		_strict = strict[0]
	}
	if err := godotenv.Load(); err != nil {
		if _strict {
			log.Fatalf("error loading env files. %v", err)
		}
		log.Printf("error loading env files. %v", err)
	}
	if err := logr.Config(); err != nil {
		logr.Log().WithError(err).Fatal("Application log initialisation failed")
	}
	logr.Log().Println("starting app initialisation")
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

func Consul() (client *consulapi.Client) {
	a := instance()
	if client = a.consul; client == nil {
		a.Log().Fatal("consul client not initialsed")
	}	
	return 
}

func Log() (*logr.Logger) {
	return instance().log
}

func Service(name string) (out *consulapi.AgentService, err error) {
	var ok bool
	if services, err := Consul().Agent().Services(); err != nil {
		return out, errors.Wrap(err, "error fetching registered consul services")
	} else if out, ok = services[name]; !ok {
		return out, errors.Errorf("service %s unknown to consul agent", name)
	}
	return
}

func ServiceUrl(name string) (out string) {
	service, err := Service(name)
	if err == nil {
		return fmt.Sprintf("http://%s:%v", service.Address, service.Port)
	}
	Log().WithError(err).WithField("consul.service", name).Error("failed to get service url")
	return
}

func (a *ap) Register(name ...string) (err error) {
	var consulUrlEnv = "CONSUL_HTTP_ADDR"
	if a.consul != nil {
		return utils.Error.LogOK(a.log.Infof, "service already registered with consul")
	} else if address := utils.Env.GetOrDefault(consulUrlEnv, ""); !address.Valid() {
		return utils.Error.LogOK(a.log.Warnf, "env var %s not set. skipping consul initialization", consulUrlEnv)
	}
	service, config := a.conf.Name, consulapi.DefaultConfig()
	if len(name) > 0 && name[0] != "" {
		service = name[0]
	}
	if a.consul, err = consulapi.NewClient(config); err != nil {
		return utils.Error.Log(a.log.Entry, err, "consul client initialisation failed")
	} else if err = a.consul.Agent().ServiceRegister(&consulapi.AgentServiceRegistration{
		ID: service,
		// ID:      fmt.Sprintf("%s-%s", a.conf.Name, a.conf.Host),
		Name:    service,
		Port:    a.conf.HttpPort,
		Address: a.conf.Host,
		Tags:    []string{a.conf.Version, a.conf.Name, a.conf.Host},
		Check: &consulapi.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://%s:%v/ok", a.conf.Host, a.conf.HttpPort),
			Interval: a.conf.Uptime.Interval.String(),
			Timeout:  a.conf.Uptime.Timeout.String(),
		},
	}); err != nil {
		return utils.Error.Log(a.log.Entry, err, "service registration failed")
	}
	return utils.Error.LogOK(a.log.Infof, "service successfully registered with consul")
}

func (a *ap) Run() {
	fmt.Println()
	utils.Error.Fail(a.log.Entry, a.Register(), "failed to register service with consul")
	signal.Notify(a.osc, os.Interrupt, syscall.SIGTERM)
	startupMsg := "started"
	if a.router != nil {
		startupMsg += " with http router " + a.conf.Address()
		go a.router.Run(a.conf.Address())
	}
	a.log.Printf(startupMsg)
	<-a.osc
	a.cancel()
	fmt.Println()
	a.shutdown()
	a.log.Printf("shutdown complete")
}

func (a *ap) shutdown() {
	a.log.Printf("shutting down")
}

func (a *ap) initAPI() {
	var (
		ok = func(c *gin.Context) {
			c.JSON(http.StatusOK, "OK")
		}
		status = func(c *gin.Context) {
			c.JSON(http.StatusOK, map[string]interface{}{
				"host": a.conf.Host, "started": a.start, "app": a.conf.Name,
				"uptime": time.Since(a.start).Round(100 * time.Millisecond).String(),
			})
		}
	)
	a.router = gin.New()
	a.router.Use(api.JSONLogMiddleware(a.log), gin.Recovery(), cors.New(cors.Config{
		AllowOrigins:     a.conf.Http.AllowOrigins,
		AllowMethods:     a.conf.Http.AllowMethods,
		AllowHeaders:     a.conf.Http.AllowHeaders,
		ExposeHeaders:    a.conf.Http.ExposeHeaders,
		AllowCredentials: a.conf.Http.AllowCredentials,
		// AllowOriginFunc: func(origin string) bool {
		// 	return true
		// },
		MaxAge: a.conf.Http.MaxAge,
	}))
	a.router.GET("/", ok)
	a.router.GET("/ok", ok)
	a.router.GET("/stats", status)
}

func instance() *ap {
	if _instance == nil {
		panic("app instance not initialised")
	}
	return _instance
}