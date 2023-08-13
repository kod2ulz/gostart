package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/object"
	"github.com/kod2ulz/gostart/utils"
)

var (
	ErrorCodeServerError             string = "ServerError"
	ErrorCodeNotFoundError           string = "NotFoundError"
	ErrorCodeIntegrationError        string = "IntegrationError"
	ErrorCodeRequestLoadError        string = "RequestLoadError"
	ErrorCodeServiceError            string = "ServiceError"
	ErrorCodeResponseProcessingError string = "ResponseProcessingError"
	ErrorCodeValidatorError          string = "ValidationError"
	ErrorCodeSQLError                string = "SQLError"
	ErrorCodeUnauthorized            string = "InvalidCredentials"
	ErrorCodeInvalidOperation        string = "InvalidOperation"
)

type Error interface {
	error
	http() int
	WithErrorCode(code string) (out Error)
	WithHttpStatusCode(code int) (out Error)
	WithErrorCodeAndHttpStatusCode(errorCode string, statusCode int) (out Error)
	WithMessage(message string) (out Error)
	WithError(err error) (out Error)
	WithCause(err Error) (out Error)
	Response() interface{}
}

type ErrorModel[T any] struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Code    string            `json:"code"`
	Http    int               `json:"status"`
	Param   RequestParam      `json:"params,omitempty"`
	Errors  []string          `json:"data,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
	Cause   Error             `json:"cause,omitempty"`
}

func (e *ErrorModel[T]) http() int {
	return e.Http
}

func (e *ErrorModel[T]) Error() string {
	return e.Message
}

func (e *ErrorModel[T]) WithErrorCode(errorCode string) (out Error) {
	e.Code = errorCode
	return e
}

func (e *ErrorModel[T]) WithHttpStatusCode(statusCode int) (out Error) {
	e.Http = statusCode
	return e
}

func (e *ErrorModel[T]) WithErrorCodeAndHttpStatusCode(errorCode string, statusCode int) (out Error) {
	return e.WithErrorCode(errorCode).WithHttpStatusCode(statusCode)
}

func (e *ErrorModel[T]) WithMessage(message string) (out Error) {
	e.Message = message
	return e
}

func (e *ErrorModel[T]) WithError(err error) (out Error) {
	if len(e.Errors) == 0 {
		e.Errors = []string{}
	}
	e.Errors = append(e.Errors, err.Error())
	return e
}

func (e *ErrorModel[T]) WithCause(err Error) (out Error) {
	e.Cause = err
	return e
}

func (e *ErrorModel[T]) Response() (out interface{}) {
	return ErrorResponse[T](e)
}

func _initError[T any](httpCode int, statusCode string, err error) (out ErrorModel[T]) {
	var message string
	var errorMessages collections.List[string]
	if err != nil {
		message = err.Error()
	}
	if message != "" && strings.Contains(message, " .") {
		errorMessages = object.String(message).Split(" .")
		message = errorMessages.Last()
	}
	out = ErrorModel[T]{
		Type:    strings.TrimPrefix(fmt.Sprintf("%T", new(T)), "*"),
		Message: message,
		Code:    statusCode,
		Http:    httpCode,
		Errors:  errorMessages,
	}
	if out.Type == "interface{}" {
		out.Type = "Undefined"
	}
	return
}

func ServerError(err error) (out Error) {
	return GeneralError[any](err)
}

func ServiceError(err error) (out Error) {
	return GeneralError[User](err).
		WithErrorCodeAndHttpStatusCode(ErrorCodeServiceError, http.StatusUnauthorized)
}

func ServiceErrorUnauthorised(err error) (out Error) {
	return GeneralError[User](err).
		WithErrorCodeAndHttpStatusCode(ErrorCodeUnauthorized, http.StatusUnauthorized)
}

func GeneralError[T any](err error) (out Error) {
	er := _initError[T](http.StatusInternalServerError, ErrorCodeServerError, err)
	if err == nil || !strings.Contains(err.Error(), ". ") {
		return &er
	}
	er.Errors = object.String(err.Error()).Split(". ").ForEach(func(i int, val string) string {
		return strings.Trim(val, "\n ")
	})
	return &er
}

func NotFoundError[T any, P RequestParam](param P) (out Error) {
	er := _initError[T](http.StatusNotFound, ErrorCodeNotFoundError, nil)
	er.Param = param
	if er.Message == "" {
		er.Message = "Not Found"
	}
	return &er
}

func RequestLoadError[T any](err error) (out Error) {
	return GeneralError[T](err).WithErrorCodeAndHttpStatusCode(ErrorCodeValidatorError, http.StatusBadRequest)
}

func ValidatorError[T any](err error) (out Error) {
	er := _initError[T](http.StatusBadRequest, ErrorCodeValidatorError, err)
	if err == nil || !strings.Contains(err.Error(), "Key:") {
		return &er
	}
	errs := strings.Split(err.Error(), "Key:")
	if len(errs) == 0 {
		return &er
	}
	er.Errors, er.Fields = make([]string, 0), map[string]string{}
	for _, msg := range errs {
		if !strings.Contains(msg, "Error:") {
			er.Errors = append(er.Errors, strings.Trim(msg, "\n :"))
		} else {
			fv := strings.Split(msg, " Error:")
			er.Fields[strings.Trim(fv[0], " '")] = strings.Trim(fv[1], " \n")
		}
	}
	if len(er.Errors) == 1 {
		er.Message = er.Errors[0]
		er.Errors = nil
	}
	return &er
}

func SQLError[T any](err error) (out Error) {
	return GeneralError[T](err).WithErrorCode(ErrorCodeSQLError)
}

func SqlQueryError[P RequestParam, T any](param P, out T, err error) (T, Error) {
	if err != nil {
		if utils.Error.SqlNoRows(err) {
			return out, NotFoundError[T](param)
		}
		return out, SQLError[T](err)
	}
	return out, nil
}
