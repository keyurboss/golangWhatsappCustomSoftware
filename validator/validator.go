package validator

import (
	"github.com/go-playground/validator/v10"
)

type (
	User struct {
		Name string `validate:"required,min=5,max=20"` // Required field, min 5 char long max 20
		Age  int    `validate:"required,teener"`       // Required field, and client needs to implement our 'teener' tag format which we'll see later
	}

	ErrorResponse struct {
		Error       bool
		FailedField string
		Tag         string
		Value       interface{}
	}

	XValidator struct {
		validator *validator.Validate
	}

	GlobalErrorHandlerResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
)

var Validator *XValidator

func init() {
	println("Validator Init")
	Validator = &XValidator{
		validator: validator.New(),
	}
	Validator.validator.RegisterValidation("port", validatePort)
}

func (v XValidator) Validate(data interface{}) []ErrorResponse {
	validationErrors := []ErrorResponse{}

	errs := v.validator.Struct(data)
	if errs != nil {
		for _, err := range errs.(validator.ValidationErrors) {
			// In this case data object is actually holding the User struct
			var elem ErrorResponse

			elem.FailedField = err.Field() // Export struct field name
			elem.Tag = err.Tag()           // Export struct tag
			elem.Value = err.Value()       // Export field value
			elem.Error = true

			validationErrors = append(validationErrors, elem)
		}
	}

	return validationErrors
}

/**
OUTPUT

[1]
Request:

GET http://127.0.0.1:3000/

Response:

{"success":false,"message":"[Name]: '' | Needs to implement 'required' and [Age]: '0' | Needs to implement 'required'"}

[2]
Request:

GET http://127.0.0.1:3000/?name=efdal&age=9

Response:
{"success":false,"message":"[Age]: '9' | Needs to implement 'teener'"}

[3]
Request:

GET http://127.0.0.1:3000/?name=efdal&age=

Response:
{"success":false,"message":"[Age]: '0' | Needs to implement 'required'"}

[4]
Request:

GET http://127.0.0.1:3000/?name=efdal&age=18

Response:
Hello, World!

**/
