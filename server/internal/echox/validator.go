package echox

import (
	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type Validator struct{}

func (Validator) Validate(v any) error {
	return validate.Struct(v)
}
