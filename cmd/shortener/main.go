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
	"github.com/anatolyi0311/cupurl/internal/app/https"
	"github.com/anatolyi0311/cupurl/internal/app/logcfg"
	"github.com/anatolyi0311/cupurl/internal/app/repositories"
	"github.com/anatolyi0311/cupurl/internal/app/server"
	"github.com/anatolyi0311/cupurl/internal/app/services"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	realip "github.com/thanhhh/gin-gonic-realip"
)

func main() {
	var (
		dbPool            *pgxpool.Pool
		err               error
		cfg               *config.ENVConfig
		myRepository      services.Repository
		repositoryReciver bool
	)

	// Выводим сообщение о сборке проекта
	config.PrintProjectInfo()

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

	server := server.NewServer(cfg, setRouters(myHandler))

	// Запуск отдельного audit
	ctxAudit, cancelAudit := context.WithCancel(context.Background())
	defer cancelAudit()
	go myHandler.RunAudit(ctxAudit)

	go func() {
		if cfg.EnvHTTPS != "" {
			_, err = https.NewHTTPS()
			if err != nil {
				logrus.Error(err)
			}
			if err = server.RunTLS(cfg.EnvServAdr); !errors.Is(err, http.ErrServerClosed) {
				logrus.Fatal(err)
			}
		} else {
			if err = server.Run(cfg.EnvServAdr); !errors.Is(err, http.ErrServerClosed) {
				logrus.Error(err)
			}
		}
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-signalChan

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = server.Stop(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "HTTP server Shutdown: %v\n", err)
	}

	//If the server shutting down, save batch to file
	if repositoryReciver {
		saveBatchToFile(myRepository)
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

func setRouters(myHandler *handlers.Handlers) *gin.Engine {
	// Установка переменной окружения для включения режима разработки
	gin.SetMode(gin.DebugMode)
	router := gin.Default()

	// Pprof роутер
	// pprofRouter := router.Group("/debug/pprof")
	// pprofRouter.Handle("GET", "/", myHandler.PprofIndex)
	// pprofRouter.GET("/", myHandler.PprofIndex)
	// pprofRouter.GET("/profile", myHandler.PprofProfile)
	// pprofRouter.GET("/goroutine", myHandler.PprofGoroutine)

	// Use the pprof middleware
	pprof.Register(router)

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

	//Only trusted subnet middleware
	trustSubnetRouter := router.Group("/")
	trustSubnetRouter.Use(realip.RealIP())
	trustSubnetRouter.Use(handlers.TrustedSubnet(myHandler.TrustedSubnets))

	trustSubnetRouter.GET("/api/internal/stats", myHandler.GetServiceStats)

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

func saveBatchToFile(myRepository services.Repository) {
	if _, ok := myRepository.(services.URLInMemoryRepository); !ok {
		logrus.Errorf("invalid type assertion %v", myRepository)
	}
	err := myRepository.(services.URLInMemoryRepository).SaveBatchToFile()
	if err != nil {
		logrus.Error(err)
	}
}
