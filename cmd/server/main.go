package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/internal/repository"
)

type CreateUrlRequest struct {
	URL       string `json:"url"`
	Shortcode string `json:"shortcode"`
}

type UpdateUrlRequest struct {
	NewURL      string `json:"url,omitempty"`
	NewShortURL string `json:"short,omitempty"`
}

func main() {
	dsnsEnv := os.Getenv("SHARD_DSNS")
	if dsnsEnv == "" {
		panic("SHARD_DSNS env not defined")
	}

	dsns := strings.Split(dsnsEnv, ",")
	shardManager, err := repository.NewShardManager(dsns)
	if err != nil {
		panic(err)
	}

	urlRepository := repository.NewURLRepository(shardManager)

	//
	// routes and API things
	//

	e := echo.New()

	e.POST("/", func(c echo.Context) error {
		var url CreateUrlRequest
		err := c.Bind(&url)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		err = urlRepository.Save(url.Shortcode, url.URL)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		return c.NoContent(http.StatusCreated)
	})

	// redirect route
	e.GET("/:shortcode", func(c echo.Context) error {
		shortcode := c.Param("shortcode")
		if shortcode == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid shortcode param to store")
		}

		url, err := urlRepository.Find(shortcode)
		if err != nil {
			return c.String(http.StatusBadRequest, err.Error())
		}

		fmt.Printf("%+v", url)

		return c.Redirect(http.StatusMovedPermanently, url)
	})

	e.Logger.Fatal(e.Start(":3030"))
}
