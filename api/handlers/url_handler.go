package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/api/dtos"
	"github.com/rodrigocitadin/url-shortener/internal/services"
)

type URLHandler struct {
	URLService services.URLService
}

func (h *URLHandler) GetFullURL(c echo.Context) error {
	var req dtos.GetUrlRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid shortcode param to store")
	}

	url, err := h.URLService.Get(req.Shortcode)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.Redirect(http.StatusMovedPermanently, url.URL)
}

func (h *URLHandler) StoreFullURL(c echo.Context) error {
	var req dtos.StoreUrlRequest
	err := c.Bind(&req)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = h.URLService.Store(req.Shortcode, req.URL)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusCreated)
}
