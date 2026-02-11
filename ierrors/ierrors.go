package ierrors

// Error is the interface for all errors in the gostart ecosystem.
type Error interface {
	error
	HttpCode() int
	WithErrorCode(code string) (out Error)
	WithHttpStatusCode(code int) (out Error)
	WithErrorCodeAndHttpStatusCode(errorCode string, statusCode int) (out Error)
	WithMessage(message string, opts ...any) (out Error)
	WithError(err error) (out Error)
	WithCause(err Error) (out Error)
	Response() interface{}
}
