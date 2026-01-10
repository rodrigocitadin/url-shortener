package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type CreateUrlRequest struct {
	URL      string `json:"url"`
	ShortURL string `json:"short"`
}

type UpdateUrlRequest struct {
	NewURL      string `json:"url,omitempty"`
	NewShortURL string `json:"short,omitempty"`
}

func main() {
	e := echo.New()

	e.POST("/", func(c echo.Context) error {
		panic("Not implemented yet")
	})

	e.PUT("/:shorturl", func(c echo.Context) error {
		shorturl, err := verifyUrlParam(c, "shorturl")
		if err != nil {
			return err
		}

		_ = shorturl

		panic("Not implemented yet")
	})

	e.DELETE("/:shorturl", func(c echo.Context) error {
		shorturl, err := verifyUrlParam(c, "shorturl")
		if err != nil {
			return err
		}

		_ = shorturl

		panic("Not implemented yet")
	})

	// redirect route
	e.GET("/:shorturl", func(c echo.Context) error {
		shorturl, err := verifyUrlParam(c, "shorturl")
		if err != nil {
			return err
		}

		// logic to find the requested shorturl
		_ = shorturl
		url := ""

		if url == "" {
			return c.String(http.StatusNotFound, "url not found")
		}

		return c.Redirect(http.StatusMovedPermanently, url)
	})

	e.Logger.Fatal(e.Start(":3030"))
}

func verifyUrlParam(c echo.Context, param string) (string, error) {
	urlParam := c.Param(param)
	if urlParam == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "invalid url param to store")
	}

	return urlParam, nil

}
