package main

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/api"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
	"github.com/rodrigocitadin/url-shortener/internal/services"
)

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
	// routes and API things
	//

	e := echo.New()
	api.Router(e, serviceChain)

	e.Logger.Fatal(e.Start(":3030"))
}
