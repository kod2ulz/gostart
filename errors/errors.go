package errors

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/kod2ulz/gostart/collections"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/object"
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

type ErrorModel[T any] struct {
	Type    string            `json:"type"`
	Message string            `json:"message"`
	Code    string            `json:"code"`
	Http    int               `json:"status"`
	Param   any               `json:"params,omitempty"`
	Errors  []string          `json:"data,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
	Cause   ierrors.Error     `json:"cause,omitempty"`
}

func (e *ErrorModel[T]) HttpCode() int {
	return e.Http
}

func (e *ErrorModel[T]) Error() string {
	return e.Message
}

func (e *ErrorModel[T]) WithErrorCode(errorCode string) (out ierrors.Error) {
	e.Code = errorCode
	return e
}

func (e *ErrorModel[T]) WithHttpStatusCode(statusCode int) (out ierrors.Error) {
	e.Http = statusCode
	return e
}

func (e *ErrorModel[T]) WithErrorCodeAndHttpStatusCode(errorCode string, statusCode int) (out ierrors.Error) {
	return e.WithErrorCode(errorCode).WithHttpStatusCode(statusCode)
}

func (e *ErrorModel[T]) WithMessage(message string, opts ...any) (out ierrors.Error) {
	e.Message = fmt.Sprintf(message, opts...)
	return e
}

func (e *ErrorModel[T]) WithError(err error) (out ierrors.Error) {
	if len(e.Errors) == 0 {
		e.Errors = []string{}
	}
	e.Errors = append(e.Errors, err.Error())
	return e
}

func (e *ErrorModel[T]) WithCause(err ierrors.Error) (out ierrors.Error) {
	e.Cause = err
	return e
}

func (e *ErrorModel[T]) Response() (out interface{}) {
	return nil
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

func ServiceFailure(err error) (out ierrors.Error) {
	return GeneralFailure[any](err).
		WithErrorCodeAndHttpStatusCode(ErrorCodeServiceError, http.StatusUnauthorized)
}

func ServiceUnauthorised(err error) (out ierrors.Error) {
	return GeneralFailure[any](err).
		WithErrorCodeAndHttpStatusCode(ErrorCodeUnauthorized, http.StatusUnauthorized)
}

func GeneralFailure[T any](err error) (out ierrors.Error) {
	er := _initError[T](http.StatusInternalServerError, ErrorCodeServerError, err)
	if err == nil || !strings.Contains(err.Error(), ". ") {
		return &er
	}
	er.Errors = object.String(err.Error()).Split(". ").ForEach(func(i int, val string) string {
		return strings.Trim(val, "\n ")
	})
	return &er
}

func NotFound[T any, P any](param P) (out ierrors.Error) {
	er := _initError[T](http.StatusNotFound, ErrorCodeNotFoundError, nil)
	er.Param = param
	if er.Message == "" {
		er.Message = "Not Found"
	}
	return &er
}

func RequestLoadFailed[T any](err error) (out ierrors.Error) {
	// Enhanced request loading error parsing
	errMsg := err.Error()
	errorModel := _initError[T](http.StatusBadRequest, ErrorCodeRequestLoadError, err)

	// Try to provide more specific error messages based on common patterns
	switch {
	case strings.Contains(errMsg, "JSON"):
		errorModel.Message = "Invalid JSON format in request body"
	case strings.Contains(errMsg, "binding"):
		errorModel.Message = "Failed to bind request data"
	case strings.Contains(errMsg, "required"):
		errorModel.Message = "Required fields are missing from request"
	case strings.Contains(errMsg, "validation"):
		errorModel.Message = "Request validation failed"
	case strings.Contains(errMsg, "unmarshal"):
		errorModel.Message = "Failed to parse request data"
	default:
		errorModel.Message = "Invalid request data"
	}

	// Try to extract field information from common error patterns
	if strings.Contains(errMsg, "field") || strings.Contains(errMsg, "Field") {
		// Basic field extraction for common error formats
		if matches := regexp.MustCompile(`field[\"']?\s*[:=]\s*[\"']?(\w+)[\"']?`).FindStringSubmatch(errMsg); len(matches) > 1 {
			errorModel.Fields = map[string]string{matches[1]: errorModel.Message}
		}
	}

	return &errorModel
}

func ValidatorError[T any](err error) (out ierrors.Error) {
	// Try to parse as go-playground/validator errors (supports multiple errors)
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		return parseValidationErrors[T](validationErrors)
	}

	// Use enhanced validation error parsing (for backward compatibility with existing error format)
	if validatorInfo := parseLegacyValidationError(err.Error()); validatorInfo != nil {
		errorModel := _initError[T](http.StatusBadRequest, ErrorCodeValidatorError, err)
		errorModel.Message = validatorInfo.Message
		if len(validatorInfo.Fields) > 0 {
			errorModel.Fields = validatorInfo.Fields
		}
		if len(validatorInfo.Details) > 0 {
			errorModel.Param = validatorInfo.Details
		}
		return &errorModel
	}

	// Fallback to basic validation error
	return GeneralFailure[T](err).WithErrorCodeAndHttpStatusCode(ErrorCodeValidatorError, http.StatusBadRequest)
}

// ValidationFailed[T any] provides enhanced validation error parsing for go-playground/validator errors
// This is the recommended function for handling validation errors from the service layer
func ValidationFailed[T any](err error) (out ierrors.Error) {
	if err == nil {
		return nil
	}

	// Try to parse as validator error first
	if validatorInfo := ParseValidatorError(wrapAsIError(err)); validatorInfo != nil {
		errorModel := _initError[T](http.StatusBadRequest, ErrorCodeValidatorError, err)
		errorModel.Message = validatorInfo.Message
		if len(validatorInfo.Fields) > 0 {
			errorModel.Fields = validatorInfo.Fields
		}
		if len(validatorInfo.Details) > 0 {
			errorModel.Param = validatorInfo.Details
		}
		return &errorModel
	}

	// Fallback to basic validation error
	return GeneralFailure[T](err).WithErrorCodeAndHttpStatusCode(ErrorCodeValidatorError, http.StatusBadRequest)
}

// wrapAsIError wraps a standard error as ierrors.Error for parsing
func wrapAsIError(err error) ierrors.Error {
	return GeneralFailure[any](err)
}

func DatabaseFailure[T any](err error) (out ierrors.Error) {
	// Use enhanced SQL error parsing
	if sqlInfo := ParseSQLError(err); sqlInfo != nil {
		// Map SQL error codes to HTTP status codes
		var httpCode = http.StatusInternalServerError
		switch sqlInfo.ErrorCode {
		case "DUPLICATE_ENTRY", "FOREIGN_KEY_VIOLATION", "CONSTRAINT_VIOLATION":
			httpCode = http.StatusConflict
		case "REQUIRED_FIELD", "VALUE_TOO_LONG", "VALUE_OUT_OF_RANGE":
			httpCode = http.StatusBadRequest
		}

		// Create the error with enhanced information
		errorModel := _initError[T](httpCode, sqlInfo.ErrorCode, err)
		errorModel.Message = sqlInfo.Message

		// Add field-level errors if available
		if sqlInfo.Column != "" {
			errorModel.Fields = map[string]string{sqlInfo.Column: sqlInfo.Message}
		}

		// Add detailed information as param
		if len(sqlInfo.Details) > 0 {
			errorModel.Param = sqlInfo.Details
		}

		return &errorModel
	}

	// Fallback to generic SQL error
	return GeneralFailure[T](err).WithErrorCode(ErrorCodeSQLError)
}

func DatabaseOperationFailed[P any, T any](param P, out T, err error) (T, ierrors.Error) {
	if err != nil {
		if DbNoRows(err) {
			return out, NotFound[T](param)
		}
		return out, DatabaseFailure[T](err) // This now uses enhanced parsing
	}
	return out, nil
}

func Wrapf(err error, format string, args ...interface{}) ierrors.Error {
	return GeneralFailure[any](fmt.Errorf(format, args...)).WithCause(GeneralFailure[any](err))
}

func Errorf(format string, args ...interface{}) ierrors.Error {
	return GeneralFailure[any](fmt.Errorf(format, args...))
}

func SqlNoRows(err error) bool {
	return DbNoRows(err)
}

func DbNoRows(err error) bool {
	return err != nil && errors.Is(err, sql.ErrNoRows) || strings.HasSuffix(err.Error(), "no rows in result set")
}

// ValidationErrorInfo represents validation error details
type ValidationErrorInfo struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err ierrors.Error) bool {
	if err == nil {
		return false
	}

	// Check if the error response contains validation error code
	if response := err.Response(); response != nil {
		if resp, ok := response.(map[string]any); ok {
			if code, ok := resp["code"].(string); ok {
				return code == ErrorCodeValidatorError
			}
		}
	}

	return false
}

// GetValidationErrorInfo extracts validation error information from an error
func GetValidationErrorInfo(err ierrors.Error) *ValidationErrorInfo {
	if !IsValidationError(err) {
		return nil
	}

	// Try to extract field information from error details
	info := &ValidationErrorInfo{
		Message: err.Error(),
	}

	// Check if the error response contains field details
	if response := err.Response(); response != nil {
		if resp, ok := response.(map[string]any); ok {
			if msg, ok := resp["message"].(string); ok {
				info.Message = msg
			}
			if fields, ok := resp["fields"].(map[string]any); ok {
				fieldMap := make(map[string]string)
				for k, v := range fields {
					if str, ok := v.(string); ok {
						fieldMap[k] = str
					}
				}
				info.Fields = fieldMap
			}
		}
	}

	return info
}

// HandleValidationError processes a validation error and returns structured information
func HandleValidationError(err ierrors.Error) (errorCode string, errorMessage string, httpCode int, fields map[string]string) {
	errorCode = ErrorCodeValidatorError
	httpCode = http.StatusBadRequest

	// Try to parse as validator error first
	if info := ParseValidatorError(err); info != nil {
		errorMessage = info.Message
		fields = info.Fields
	} else if info := GetValidationErrorInfo(err); info != nil {
		errorMessage = info.Message
		fields = info.Fields
	} else {
		errorMessage = err.Error()
	}

	return errorCode, errorMessage, httpCode, fields
}

// ValidatorErrorInfo represents detailed validation error information
type ValidatorErrorInfo struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details map[string]any    `json:"details,omitempty"`
}

// ParseValidatorError attempts to parse go-playground/validator errors
func ParseValidatorError(err ierrors.Error) *ValidatorErrorInfo {
	if err == nil {
		return nil
	}

	// Get the underlying error message
	errMsg := err.Error()

	// Check if it's wrapped validator error (old format)
	if strings.Contains(errMsg, "Key:") && strings.Contains(errMsg, "Error:") {
		return parseLegacyValidationError(errMsg)
	}

	return nil
}

// parseLegacyValidationError parses the old format validation errors
func parseLegacyValidationError(errMsg string) *ValidatorErrorInfo {
	fields := make(map[string]string)
	details := make(map[string]any)
	errorMessages := make([]string, 0)

	// Split by "Key:" to get individual errors
	errorParts := strings.Split(errMsg, "Key:")

	for _, part := range errorParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Extract field name and error message
		if strings.Contains(part, "Error:") {
			fieldError := strings.SplitN(part, "Error:", 2)
			if len(fieldError) == 2 {
				fieldName := strings.TrimSpace(fieldError[0])
				errorMsg := strings.TrimSpace(fieldError[1])

				fields[fieldName] = errorMsg
				errorMessages = append(errorMessages, errorMsg)

				// Try to parse rule from error message
				if rule := extractRuleFromMessage(errorMsg); rule != "" {
					details[fieldName] = map[string]any{"rule": rule, "message": errorMsg}
				}
			}
		}
	}

	// Create summary message
	var summaryMessage string
	if len(errorMessages) == 1 {
		summaryMessage = errorMessages[0]
	} else {
		summaryMessage = fmt.Sprintf("Validation failed for %d fields", len(fields))
	}

	return &ValidatorErrorInfo{
		Message: summaryMessage,
		Fields:  fields,
		Details: details,
	}
}

// getValidationMessage generates user-friendly validation messages
func getValidationMessage(field, rule, param string) string {
	switch rule {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		if num, err := strconv.Atoi(param); err == nil {
			return fmt.Sprintf("%s must be at least %d characters", field, num)
		}
		return fmt.Sprintf("%s must meet minimum requirements", field)
	case "max":
		if num, err := strconv.Atoi(param); err == nil {
			return fmt.Sprintf("%s must be at most %d characters", field, num)
		}
		return fmt.Sprintf("%s exceeds maximum allowed length", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "url":
		return fmt.Sprintf("%s must be a valid URL", field)
	case "numeric":
		return fmt.Sprintf("%s must contain only numbers", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.ReplaceAll(param, " ", ", "))
	case "unique":
		return fmt.Sprintf("%s must be unique", field)
	default:
		// Default message for unknown validation rules
		return fmt.Sprintf("%s failed validation: %s", field, rule)
	}
}

// parseValidationErrors parses go-playground/validator ValidationErrors into a ValidatorErrorInfo
// This function properly handles multiple validation errors and uses JSON tag names
func parseValidationErrors[T any](validationErrors validator.ValidationErrors) ierrors.Error {
	fields := make(map[string]string)
	details := make(map[string]any)
	errorMessages := make([]string, 0, len(validationErrors))

	// Iterate through all validation errors
	for _, err := range validationErrors {
		fieldName := err.Field()      // JSON tag name (thanks to RegisterTagNameFunc)
		rule := err.Tag()             // Validation rule that failed
		param := err.Param()          // Parameter for the rule (e.g., min length)

		// Generate user-friendly error message
		errorMsg := getValidationMessage(fieldName, rule, param)
		fields[fieldName] = errorMsg
		errorMessages = append(errorMessages, errorMsg)

		// Add detailed information
		details[fieldName] = map[string]any{
			"rule":    rule,
			"message": errorMsg,
			"param":   param,
		}
	}

	// Create summary message
	var summaryMessage string
	if len(errorMessages) == 1 {
		summaryMessage = errorMessages[0]
	} else {
		summaryMessage = fmt.Sprintf("Validation failed for %d field(s)", len(fields))
	}

	// Create error model
	errorModel := _initError[T](http.StatusBadRequest, ErrorCodeValidatorError, validationErrors)
	errorModel.Message = summaryMessage
	if len(fields) > 0 {
		errorModel.Fields = fields
	}
	if len(details) > 0 {
		errorModel.Param = details
	}

	return &errorModel
}

// extractRuleFromMessage attempts to extract validation rule from error message
func extractRuleFromMessage(message string) string {
	// Common patterns in validator error messages
	patterns := map[string]*regexp.Regexp{
		"required": regexp.MustCompile(`(?i)required|missing`),
		"min":      regexp.MustCompile(`(?i)minimum|min.*at least`),
		"max":      regexp.MustCompile(`(?i)maximum|max.*at most`),
		"email":    regexp.MustCompile(`(?i)email.*valid`),
		"numeric":  regexp.MustCompile(`(?i)numeric|number`),
		"length":   regexp.MustCompile(`(?i)length|characters`),
		"greater":  regexp.MustCompile(`(?i)greater.*than`),
		"less":     regexp.MustCompile(`(?i)less.*than`),
	}

	for rule, pattern := range patterns {
		if pattern.MatchString(message) {
			return rule
		}
	}

	return ""
}

// SQLErrorInfo represents detailed SQL error information
type SQLErrorInfo struct {
	Message    string            `json:"message"`
	ErrorCode  string            `json:"error_code"`
	Constraint string            `json:"constraint,omitempty"`
	Table      string            `json:"table,omitempty"`
	Column     string            `json:"column,omitempty"`
	Fields     map[string]string `json:"fields,omitempty"`
	Details    map[string]any    `json:"details,omitempty"`
}

// ParseSQLError attempts to parse database errors and extract meaningful information
func ParseSQLError(err error) *SQLErrorInfo {
	if err == nil {
		return nil
	}

	// Handle PostgreSQL errors specifically
	if strings.Contains(err.Error(), "pq:") || strings.Contains(err.Error(), "ERROR:") {
		return parsePostgresError(err.Error())
	}

	// Handle common MySQL errors
	if strings.Contains(err.Error(), "Error 1062") {
		return parseMySQLDuplicateError(err.Error())
	}

	// Handle SQLite errors
	if strings.Contains(err.Error(), "UNIQUE constraint failed") {
		return parseSQLiteDuplicateError(err.Error())
	}

	// Generic SQL error parsing
	return parseGenericSQLError(err.Error())
}

// parsePostgresError parses PostgreSQL error messages
func parsePostgresError(errMsg string) *SQLErrorInfo {
	info := &SQLErrorInfo{
		Message:   errMsg,
		ErrorCode: "UNKNOWN",
		Details:   make(map[string]any),
	}

	// Extract PostgreSQL error code and details
	if matches := regexp.MustCompile(`ERROR: (.+?) \(SQLSTATE (\w+)\)`).FindStringSubmatch(errMsg); len(matches) > 2 {
		info.Message = matches[1]
		info.ErrorCode = matches[2]
	}

	// Parse specific constraint violations
	switch info.ErrorCode {
	case "23505": // unique_violation
		if constraint, table, column := parseUniqueViolation(errMsg); constraint != "" {
			info.Constraint = constraint
			info.Table = table
			info.Column = column
			info.ErrorCode = "DUPLICATE_ENTRY"
			info.Message = fmt.Sprintf("A record with this %s already exists", fieldToHuman(column))
		}
	case "23503": // foreign_key_violation
		if constraint, table, column := parseForeignKeyViolation(errMsg); constraint != "" {
			info.Constraint = constraint
			info.Table = table
			info.Column = column
			info.ErrorCode = "FOREIGN_KEY_VIOLATION"
			info.Message = fmt.Sprintf("Referenced %s does not exist", fieldToHuman(column))
		}
	case "23502": // not_null_violation
		if column := parseNotNullViolation(errMsg); column != "" {
			info.Column = column
			info.ErrorCode = "REQUIRED_FIELD"
			info.Message = fmt.Sprintf("%s is required", fieldToHuman(column))
		}
	case "22001": // string_data_right_truncation
		if column := parseStringTruncation(errMsg); column != "" {
			info.Column = column
			info.ErrorCode = "VALUE_TOO_LONG"
			info.Message = fmt.Sprintf("%s exceeds maximum allowed length", fieldToHuman(column))
		}
	case "22003": // numeric_value_out_of_range
		if column := parseNumericOutOfRange(errMsg); column != "" {
			info.Column = column
			info.ErrorCode = "VALUE_OUT_OF_RANGE"
			info.Message = fmt.Sprintf("%s value is out of allowed range", fieldToHuman(column))
		}
	}

	return info
}

// parseMySQLDuplicateError parses MySQL duplicate entry errors
func parseMySQLDuplicateError(errMsg string) *SQLErrorInfo {
	info := &SQLErrorInfo{
		Message:   errMsg,
		ErrorCode: "DUPLICATE_ENTRY",
	}

	// Extract duplicate key information
	if matches := regexp.MustCompile(`Duplicate entry '(.+?)' for key '(.+?)'`).FindStringSubmatch(errMsg); len(matches) > 2 {
		info.Details = map[string]any{
			"duplicate_value": matches[1],
			"key_name":        matches[2],
		}
	}

	return info
}

// parseSQLiteDuplicateError parses SQLite unique constraint errors
func parseSQLiteDuplicateError(errMsg string) *SQLErrorInfo {
	info := &SQLErrorInfo{
		Message:   errMsg,
		ErrorCode: "DUPLICATE_ENTRY",
	}

	// Extract constraint and column info
	if matches := regexp.MustCompile(`UNIQUE constraint failed: (.+?)\.(.+?)`).FindStringSubmatch(errMsg); len(matches) > 2 {
		info.Table = matches[1]
		info.Column = matches[2]
		info.Message = fmt.Sprintf("A record with this %s already exists", fieldToHuman(matches[2]))
	}

	return info
}

// parseGenericSQLError handles generic SQL errors
func parseGenericSQLError(errMsg string) *SQLErrorInfo {
	info := &SQLErrorInfo{
		Message:   errMsg,
		ErrorCode: "SQL_ERROR",
	}

	// Try to extract common patterns
	if strings.Contains(errMsg, "duplicate") || strings.Contains(errMsg, "Duplicate") {
		info.ErrorCode = "DUPLICATE_ENTRY"
		info.Message = "A record with these values already exists"
	} else if strings.Contains(errMsg, "foreign key") || strings.Contains(errMsg, "constraint") {
		info.ErrorCode = "CONSTRAINT_VIOLATION"
	}

	return info
}

// Helper functions for parsing specific error patterns
func parseUniqueViolation(errMsg string) (constraint, table, column string) {
	// Pattern: ERROR: duplicate key value violates unique constraint "constraint_name" DETAIL: Key (column)=(value) already exists.
	if matches := regexp.MustCompile(`unique constraint "(\w+)"`).FindStringSubmatch(errMsg); len(matches) > 1 {
		constraint = matches[1]
	}
	if matches := regexp.MustCompile(`Key \((\w+)\)=`).FindStringSubmatch(errMsg); len(matches) > 1 {
		column = matches[1]
	}
	return constraint, "", column
}

func parseForeignKeyViolation(errMsg string) (constraint, table, column string) {
	// Pattern: ERROR: insert or update on table "table_name" violates foreign key constraint "constraint_name"
	if matches := regexp.MustCompile(`foreign key constraint "(\w+)"`).FindStringSubmatch(errMsg); len(matches) > 1 {
		constraint = matches[1]
	}
	if matches := regexp.MustCompile(`table "(\w+)"`).FindStringSubmatch(errMsg); len(matches) > 1 {
		table = matches[1]
	}
	return constraint, table, ""
}

func parseNotNullViolation(errMsg string) string {
	// Pattern: ERROR: null value in column "column_name" violates not-null constraint
	if matches := regexp.MustCompile(`column "(\w+)"`).FindStringSubmatch(errMsg); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func parseStringTruncation(errMsg string) string {
	// Pattern: ERROR: value too long for type character varying(n)
	if matches := regexp.MustCompile(`character varying\(\d+\)`).FindStringSubmatch(errMsg); len(matches) > 0 {
		// Extract field name from context if available
		if fieldMatches := regexp.MustCompile(`column "(\w+)"`).FindStringSubmatch(errMsg); len(fieldMatches) > 1 {
			return fieldMatches[1]
		}
	}
	return ""
}

func parseNumericOutOfRange(errMsg string) string {
	// Pattern: ERROR: numeric field overflow
	if matches := regexp.MustCompile(`column "(\w+)"`).FindStringSubmatch(errMsg); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// fieldToHuman converts database field names to human-readable format
func fieldToHuman(field string) string {
	if field == "" {
		return "field"
	}

	// Convert snake_case to Title Case
	human := strings.ReplaceAll(field, "_", " ")
	words := strings.Fields(human)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	human = strings.Join(words, " ")

	// Handle special cases
	switch human {
	case "Id":
		return "ID"
	case "Uuid":
		return "UUID"
	case "Url":
		return "URL"
	default:
		return human
	}
}

// HandleAPIError provides centralized error handling for API responses
// This function processes ierrors.Error and returns structured error information
// suitable for API response envelopes
func HandleAPIError(err ierrors.Error) (errorCode string, errorMessage string, httpCode int, fields map[string]string) {
	if err == nil {
		return "INTERNAL_ERROR", "Internal server error", http.StatusInternalServerError, nil
	}

	// Handle validation errors using the enhanced validation error parsing
	if IsValidationError(err) {
		return HandleValidationError(err)
	}

	// Handle other API errors - extract error details from response or use defaults
	errorCode = "INTERNAL_ERROR"
	errorMessage = "Internal server error"
	httpCode = http.StatusInternalServerError

	// Use the error's HTTP status code if available
	if err.HttpCode() > 0 {
		httpCode = err.HttpCode()
	}

	// Try to extract error details from response
	if response := err.Response(); response != nil {
		if resp, ok := response.(map[string]any); ok {
			if code, ok := resp["code"].(string); ok {
				errorCode = code
			}
			if msg, ok := resp["message"].(string); ok {
				errorMessage = msg
			}
		}
	}

	// Use the error message as fallback
	if errorMessage == "Internal server error" && err.Error() != "" {
		errorMessage = err.Error()
	}

	return errorCode, errorMessage, httpCode, nil
}

// EnhancedSQLError creates an enhanced SQL error with detailed parsing
func EnhancedSQLError[T any](err error, param T) (out T, apiErr ierrors.Error) {
	if err == nil {
		return out, nil
	}

	// Check for no rows error first
	if DbNoRows(err) {
		return out, NotFound[T](param)
	}

	// Try to parse the SQL error for better error messages
	if sqlInfo := ParseSQLError(err); sqlInfo != nil {
		// Create a more user-friendly error message
		var errorMessage = sqlInfo.Message
		var fields map[string]string

		if sqlInfo.Column != "" {
			fields = map[string]string{sqlInfo.Column: errorMessage}
		}

		// Map SQL error codes to HTTP status codes
		var httpCode = http.StatusInternalServerError
		switch sqlInfo.ErrorCode {
		case "DUPLICATE_ENTRY", "FOREIGN_KEY_VIOLATION", "CONSTRAINT_VIOLATION":
			httpCode = http.StatusConflict
		case "REQUIRED_FIELD", "VALUE_TOO_LONG", "VALUE_OUT_OF_RANGE":
			httpCode = http.StatusBadRequest
		}

		// Create the error with enhanced information
		errorModel := _initError[T](httpCode, sqlInfo.ErrorCode, err)
		errorModel.Message = errorMessage
		if len(fields) > 0 {
			errorModel.Fields = fields
		}
		if len(sqlInfo.Details) > 0 {
			errorModel.Param = sqlInfo.Details
		}

		return out, &errorModel
	}

	// Fallback to generic SQL error
	return out, DatabaseFailure[T](err)
}
