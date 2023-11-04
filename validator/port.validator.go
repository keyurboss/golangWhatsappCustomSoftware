package validator

import (
	"github.com/go-playground/validator/v10"
)

func validatePort(fl validator.FieldLevel) bool {
	port := fl.Field().Int()
	if port < 1 || port > 65535 {
		return false
	}
	return true
}
