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

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type ap struct {
	router *gin.Engine
	log    *logr.Logger
	start  time.Time
	conf   *conf

	osc    chan os.Signal
	ctx    context.Context
	cancel context.CancelFunc
}

func Init(strict ...bool) (a *ap) {
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
	a = &ap{
		log:   logr.Log(),
		start: time.Now(),
		conf:  Conf(),
		osc:   make(chan os.Signal, 1),
	}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.initAPI()
	return
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

func (a *ap) Run() {
	fmt.Println()
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
	a.router.GET("/check", ok)
	a.router.GET("/stats", status)
}
