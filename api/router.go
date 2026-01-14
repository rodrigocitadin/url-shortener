package api

import (
	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/api/handlers"
)

func Router(e *echo.Echo) {
	e.POST("/", handlers.StoreFullURL)
	e.GET("/:shortcode", handlers.GetFullURL)
}
