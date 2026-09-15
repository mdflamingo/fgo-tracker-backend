package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var GlobalValidator *validator.Validate

func Init() {
	GlobalValidator = validator.New()

	GlobalValidator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

func FormatValidationError(err error) map[string]string {
	errors := make(map[string]string)

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors[e.Field()] = getErrorMessage(e)
		}
	}
	return errors
}

func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return "Field required"
	case "max":
		return "The maximum length has been exceeded (" + e.Param() + " symbols)"
	case "min":
		return "The minimum length has not been reached (" + e.Param() + " symbols)"
	case "uuid":
		return "Invalid UUID"
	case "oneof":
		return "The value must be one of: " + e.Param()
	case "url":
		return "Invalid link (URL)"
	default:
		return "Invalid value"
	}
}
