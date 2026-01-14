package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/api/dtos"
	"github.com/rodrigocitadin/url-shortener/internal/services"
)

type URLHandler interface {
	GetFullURL(e echo.Context) error
	StoreFullURL(e echo.Context) error
}

type urlHandler struct {
	URLService services.URLService
}

func (h *urlHandler) GetFullURL(e echo.Context) error {
	var req dtos.GetUrlRequest
	if err := e.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid shortcode param to store")
	}

	url, err := h.URLService.Get(req.Shortcode)
	if err != nil {
		return e.String(http.StatusBadRequest, err.Error())
	}

	return e.Redirect(http.StatusMovedPermanently, url.URL)
}

func (h *urlHandler) StoreFullURL(e echo.Context) error {
	var req dtos.StoreUrlRequest
	err := e.Bind(&req)
	if err != nil {
		return e.String(http.StatusBadRequest, err.Error())
	}

	err = h.URLService.Store(req.URL, req.Shortcode)
	if err != nil {
		return e.String(http.StatusBadRequest, err.Error())
	}

	return e.NoContent(http.StatusCreated)
}

func NewURLHandler(urlService services.URLService) URLHandler {
	return &urlHandler{URLService: urlService}
}
