package utils

import (
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
	Validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		pass, ok := fl.Field().Interface().(string)
		return ok && Validator.PasswordValid(pass)
	})
}

type optionalVar interface {
	IsSet() bool
}

type nullableVar interface {
	Valid() bool
}

type validatorUtil struct {}

var Validator validatorUtil

func (validatorUtil) IsSetAny(opts ...optionalVar) bool {
	if len(opts) == 0 {
		return false
	}
	for i := range opts {
		if opts[i].IsSet() {
			return true
		}
	}
	return false
}

func (validatorUtil) IsSetAll(opts ...optionalVar) bool {
	if len(opts) == 0 {
		return false
	}
	for i := range opts {
		if !opts[i].IsSet() {
			return false
		}
	}
	return true
}

func (validatorUtil) ValidAny(vars ...nullableVar) bool {
	if len(vars) == 0 {
		return false
	}
	for i := range vars {
		if vars[i].Valid() {
			return true
		}
	}
	return false
}

func (validatorUtil) ValidAll(vars ...nullableVar) bool {
	if len(vars) == 0 {
		return false
	}
	for i := range vars {
		if !vars[i].Valid() {
			return false
		}
	}
	return true
}

func (validatorUtil) UuidAnyValid(vars ...uuid.NullUUID) bool {
	if len(vars) == 0 {
		return false
	}
	for i := range vars {
		if vars[i].Valid {
			return true
		}
	}
	return false
}

func (validatorUtil) PasswordValid(password string) bool {
	var (
		upp, low, num, sym bool
		total                uint8
	)
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			upp = true
			total++
		case unicode.IsLower(char):
			low = true
			total++
		case unicode.IsNumber(char):
			num = true
			total++
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			sym = true
			total++
		default:
			return false
		}
	}

	if !upp || !low || !num || !sym || total < 8 {
		return false
	}

	return true
}

