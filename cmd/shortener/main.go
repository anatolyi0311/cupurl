package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"

	// "runtime"
	// rpprf "runtime/pprof"
	"syscall"
	"time"

	"github.com/anatolyi0311/cupurl/internal/app/config"
	"github.com/anatolyi0311/cupurl/internal/app/handlers"
	"github.com/anatolyi0311/cupurl/internal/app/logcfg"
	"github.com/anatolyi0311/cupurl/internal/app/repositories"
	"github.com/anatolyi0311/cupurl/internal/app/services"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func main() {
	var (
		dbPool            *pgxpool.Pool
		err               error
		cfg               *config.ENVConfig
		myRepository      services.Repository
		repositoryReciver bool
	)

	cfg = config.NewConfig()
	if cfg.EnvDataBase != "" {
		dbPool = getDbPool(cfg)
		defer dbPool.Close()
		myRepository = repositories.NewURLInDBRepo(dbPool)
	} else {
		myRepository = repositories.NewURLInMemoryRepo(cfg.EnvStoragePath)
		repositoryReciver = true
	}

	logcfg.RunLoggerConfig(cfg.EnvLogLevel)
	logrus.Infof("Server started:\nServer addres %s\nBase URL %s\nFile path %s\nDBConfig %s\n",
		cfg.EnvServAdr, cfg.EnvBaseURL, cfg.EnvStoragePath, cfg.EnvDataBase)

	myShorURLService := services.NewShortURLServices(myRepository, services.ShortURLServices{}, cfg.EnvBaseURL)
	myHandler := handlers.NewHandlers(myShorURLService, dbPool, cfg)

	router := gin.Default()

	server := &http.Server{Addr: cfg.EnvServAdr, Handler: setRouters(router, myHandler)}

	// Запуск отдельного audit
	ctxAudit, cancelAudit := context.WithCancel(context.Background())
	defer cancelAudit()
	go myHandler.RunAudit(ctxAudit)

	go func() {
		logrus.Info("Starting server on: ", cfg.EnvServAdr)
		if err = server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logrus.Error(err)
		}
	}()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logrus.Info("Shutting down server...")
	if err = server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "HTTP server Shutdown: %v\n", err)
	}

	//If the server shutting down, save batch to file
	if repositoryReciver {
		if _, ok := myRepository.(services.URLInMemoryRepository); !ok {
			logrus.Errorf("invalid type assertion %v", myRepository)
		}
		err = myRepository.(services.URLInMemoryRepository).SaveBatchToFile()
		if err != nil {
			logrus.Error(err)
		}
	}

	// Запуск отдельного pprof-сервера
	// создаём файл журнала профилирования памяти
	makePprof()

	logrus.Info("Server exited")
}

func getDbPool(cfg *config.ENVConfig) *pgxpool.Pool {
	confPool, err := pgxpool.ParseConfig(cfg.EnvDataBase)
	if err != nil {
		logrus.Errorf("error parsing config: %v", err)
	}
	confPool.MaxConns = 50
	confPool.MinConns = 10
	dbPool, err := pgxpool.NewWithConfig(context.Background(), confPool)
	if err != nil {
		logrus.Error("Don't connect to DB: ", err)
		os.Exit(1)
	}
	return dbPool
}

func setRouters(router *gin.Engine, myHandler *handlers.Handlers) *gin.Engine {
	// Pprof роутер
	pprofRouter := router.Group("/debug/pprof")
	// pprofRouter.Handle("GET", "/", myHandler.PprofIndex)
	pprofRouter.GET("/", myHandler.PprofIndex)
	pprofRouter.GET("/profile", myHandler.PprofProfile)
	pprofRouter.GET("/goroutine", myHandler.PprofGoroutine)

	//Public middleware routers group
	publicRoutes := router.Group("/")
	publicRoutes.Use(myHandler.MiddlewareAuthPublic())
	publicRoutes.Use(myHandler.MiddlewareLogging())
	publicRoutes.Use(myHandler.MiddlewareCompress())

	publicRoutes.POST("/", myHandler.GetShortURL)
	publicRoutes.GET("/ping", myHandler.PingDB)
	publicRoutes.GET("/:id", myHandler.GetOriginalURL)
	publicRoutes.POST("/api/shorten", myHandler.GetJSONShortURL)
	publicRoutes.POST("/api/shorten/batch", myHandler.GetBatchShortURL)

	//Private middleware routers group
	privateRoutes := router.Group("/")
	privateRoutes.Use(myHandler.MiddlewareAuthPrivate())
	privateRoutes.Use(myHandler.MiddlewareLogging())
	privateRoutes.Use(myHandler.MiddlewareCompress())

	privateRoutes.GET("/api/user/urls", myHandler.GetUserURLS)
	privateRoutes.DELETE("/api/user/urls", myHandler.DelUserURLS)

	return router
}

func makePprof() error {
	// go func() {
	// 	_ = http.ListenAndServe("localhost:6060", nil)
	// }()
	// fmem, err := os.Create(`result.pprof`)
	// if err != nil {
	// 	panic(err)
	// }
	// defer fmem.Close()
	// runtime.GC() // получаем статистику по использованию памяти
	// if err := rpprf.WriteHeapProfile(fmem); err != nil {
	// 	panic(err)
	// }
	// err = os.Remove(`result.pprof`)
	// if err != nil {
	// 	panic(err)
	// }
	return nil
}
