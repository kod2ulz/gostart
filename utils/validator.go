package utils

import (
	"regexp"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/object"
)

var Validate *validator.Validate = validator.New()

func init() {
	Validate.RegisterValidation("password", func(fl validator.FieldLevel) bool {
		pass, ok := fl.Field().Interface().(string)
		return ok && Validator.PasswordValid(pass)
	})
	Validator.SetPhoneRegex("ug", `^\+?(0|256)(20|31|32|39|70|71|72|73|74|75|76|77|78)[0-9]{7}$`)
}

type optionalVar interface {
	IsSet() bool
}

type nullableVar interface {
	Valid() bool
}

type validatorUtil struct {
	Phone map[string]*regexp.Regexp
}

func (u *validatorUtil) SetPhoneRegex(countryCode, regexStr string) {
	var err error
	var validateFn func(fl validator.FieldLevel) bool
	if u.Phone == nil {
		u.Phone = make(map[string]*regexp.Regexp)
	}
	if u.Phone[countryCode], err = regexp.Compile(regexStr); err != nil {
		return
	} else if Validate == nil {
		return
	}
	validateFn = func(fl validator.FieldLevel) bool {
		phone, ok := fl.Field().Interface().(string)
		return ok && u.Phone[countryCode].MatchString(phone)
	}
	for _, rule := range object.String(countryCode).Variations("phone-%s", "phone_%s") {
		Validate.RegisterValidation(rule, validateFn)
	}
}

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
		total              uint8
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
