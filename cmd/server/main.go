package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rodrigocitadin/url-shortener/api"
	"github.com/rodrigocitadin/url-shortener/api/handlers"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
	"github.com/rodrigocitadin/url-shortener/internal/services"
)

var loggerConfig = middleware.RequestLoggerConfig{
	LogStatus:    true,
	LogURI:       true,
	LogMethod:    true,
	LogError:     true,
	LogRequestID: true,
	HandleError:  true,
	LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
		errStr := ""
		if v.Error != nil {
			errStr = v.Error.Error()
		}

		log.Printf("time=%s id=%s method=%s uri=%s status=%d error=%s\n",
			v.StartTime.Format(time.RFC3339),
			v.RequestID,
			v.Method,
			v.URI,
			v.Status,
			errStr,
		)
		return nil
	},
}

func main() {
	dsnsEnv := os.Getenv("SHARD_DSNS")
	if dsnsEnv == "" {
		panic("SHARD_DSNS env not defined")
	}

	dsns := strings.Split(dsnsEnv, ",")

	sm, err := repository.NewShardManager(dsns)
	if err != nil {
		panic(err)
	}

	uow := repository.NewUnitOfWork(sm)
	urlService := services.NewURLService(uow)
	serviceChain := api.ServiceChain{URLService: urlService}

	//
	// routes and API
	//

	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = handlers.HTTPErrorHandler

	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(loggerConfig))
	e.Use(middleware.Recover())

	api.Router(e, serviceChain)

	e.Logger.Fatal(e.Start(":3030"))
}
