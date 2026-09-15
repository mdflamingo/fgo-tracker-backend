package model

import (
	"net/http"

	"github.com/go-chi/render"
)

type ResponseError struct {
	ErrorMsg string `json:"error"`
	Code     int    `json:"code"`
}

func (e *ResponseError) Render(r *http.Request) error {
	return nil
}

func ResponseWithError(w http.ResponseWriter, r *http.Request, code int, errMsg string) {
	render.Status(r, code)
	render.JSON(w, r, &ResponseError{
		ErrorMsg: errMsg,
		Code:     code,
	})
}

func ResponseWithValidationError(w http.ResponseWriter, r *http.Request, details interface{}) {
	render.Status(r, http.StatusBadRequest)
	render.JSON(w, r, &ResponseError{
		ErrorMsg: "Validation failed",
		Code:     http.StatusBadRequest,
	})
}
