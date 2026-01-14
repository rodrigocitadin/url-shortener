package api

import (
	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/api/handlers"
	"github.com/rodrigocitadin/url-shortener/internal/services"
)

type ServiceChain struct {
	URLService services.URLService
}

func Router(e *echo.Echo, serviceChain ServiceChain) {
	urlHandler := handlers.NewURLHandler(serviceChain.URLService)

	e.POST("/", urlHandler.StoreFullURL)
	e.GET("/:shortcode", urlHandler.GetFullURL)
}
