package handlers

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/internal/errors"
)

func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	reqID := c.Response().Header().Get(echo.HeaderXRequestID)

	code := http.StatusInternalServerError
	response := &errors.Error{
		Code:    code,
		Message: "Internal Server Error",
		Layer:   "unknown",
	}

	if appErr, ok := err.(*errors.Error); ok {
		code = appErr.Code
		response = appErr
	} else if echoErr, ok := err.(*echo.HTTPError); ok {
		code = echoErr.Code
		response.Code = code
		response.Message = echoErr.Message.(string)
		response.Layer = "framework"
	} else {
		response.Internal = err
	}

	if code >= 500 {
		log.Printf("[%s] CRITICAL [%s]: %s | Root Cause: %v", reqID, response.Layer, response.Message, response.Internal)
	} else {
		log.Printf("[%s] INFO [%s]: %s", reqID, response.Layer, response.Message)
	}

	if err := c.JSON(code, response); err != nil {
		log.Printf("[%s] Error sending JSON: %v", reqID, err)
	}
}
