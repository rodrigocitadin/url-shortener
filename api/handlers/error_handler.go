package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rodrigocitadin/url-shortener/internal/errors"
)

func HTTPErrorHandler(err error, c echo.Context) {
	reqID := c.Response().Header().Get(echo.HeaderXRequestID)
	if reqID == "" {
		reqID = "unknown"
	}

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
		response.Message = fmt.Sprintf("%v", echoErr.Message)
		response.Layer = "framework"
	} else {
		response.Internal = err
		response.Layer = "unhandled"
	}

	if response.Internal != nil {
		log.Printf("[%s] Critial error [%s]: %s | Root Cause: %v", reqID, response.Layer, response.Message, response.Internal)
	} else {
		log.Printf("[%s] Error [%s]: %s", reqID, response.Layer, response.Message)
	}

	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, response)
		}
		if err != nil {
			log.Printf("[%s] Error sending the Error JSON: %v", reqID, err)
		}
	}
}
