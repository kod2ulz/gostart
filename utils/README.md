# Utils Package

This package provides a comprehensive collection of utility functions that extend Go's standard library with commonly needed functionality. It offers production-ready utilities for validation, formatting, mathematical operations, and more.

## Overview

The utils package is designed to eliminate repetitive boilerplate code and provide reliable, well-tested implementations of common operations:

- **String Utilities**: Phone validation, text formatting, email validation
- **Numeric Operations**: Safe mathematics, currency formatting, financial calculations
- **Time Utilities**: Time zone handling, business hours, date formatting
- **Validation**: Struct validation, custom rules, data integrity checks
- **UUID Generation**: Various UUID versions and business identifier generation
- **Performance**: Memory-efficient operations with proper error handling

## Basic Usage

The package provides intuitive APIs for common operations:

```go
// Phone number validation and formatting
phone := "+256772123456"
isValid := utils.String.IsValidPhoneNumber(phone)     // true
formatted := utils.String.FormatPhoneNumber(phone)    // "+256 772 123 456"

// Email validation
email := "user@example.com"
isValidEmail := utils.String.IsEmail(email)            // true

// Safe mathematical operations
result := utils.Math.SafeAdd(largeNumber, 1)        // Handles overflow
percentage := utils.Math.Percentage(75, 100)         // 75.0

// Time operations
now := utils.Time.NowInLocation("Africa/Kampala")
formatted := utils.Time.FormatLocal(now)              // "25-Dec-2024"
```

## String Utilities

### Phone Number Processing
```go
// Phone number validation and formatting
phoneNumber := "0772123456"

// Basic validation and formatting
isValid := utils.String.IsValidPhoneNumber(phoneNumber)   // true
formatted := utils.String.FormatPhoneNumber(phoneNumber)  // "+256 772 123 456"
normalized := utils.String.NormalizePhone(phoneNumber)   // "256772123456"

// Type detection
isMobile := utils.String.IsMobileNumber(phoneNumber)     // true
isLandline := utils.String.IsLandlineNumber(phoneNumber) // false
```

### Text Processing
```go
// Text cleaning and formatting
name := "john doe smith"
properCase := utils.String.ToProperCase(name)        // "John Doe Smith"
titleCase := utils.String.ToTitleCase(name)        // "John Doe Smith"
cleanName := utils.String.Clean(name)              // "john doe smith"

// String generation
voucherCode := utils.String.RandomAlphanumeric(12)  // "A7X9K2M4N5P1"
reference := utils.String.GenerateReference("TXN") // "TXN-2024-123456"
invoiceNo := utils.String.GenerateInvoiceNumber()    // "INV-2024-000123"
```

### Email and Text Validation
```go
// Email validation
email := "user@example.com"
isValidEmail := utils.String.IsEmail(email)        // true

// String extraction
text := "Contact us at support@example.com or call +256772123456"
emails := utils.String.ExtractEmails(text)       // ["support@example.com"]
phones := utils.String.ExtractPhoneNumbers(text) // ["+256772123456"]
urls := utils.String.ExtractURLs(text)           // [] (if no URLs)

// Case conversion
camelCase := utils.String.ToCamelCase("hello_world") // "helloWorld"
snakeCase := utils.String.ToSnakeCase("HelloWorld")   // "hello_world"
```

## Numeric Utilities

### Safe Mathematical Operations
```go
// Overflow-safe calculations
largeNumber := math.MaxInt64
result := utils.Math.SafeAdd(largeNumber, 1) // Handles overflow gracefully
safeProduct := utils.Math.SafeMultiply(1000000, 1000000)  // Safe multiplication

// Percentage calculations
percentage := utils.Math.Percentage(75, 100)    // 75.0
increase := utils.Math.PercentageIncrease(100, 120) // 20.0
discount := utils.Math.CalculateDiscount(50000, 10) // 5000 (10% of 50000)

// Statistical operations
numbers := []float64{10, 20, 30, 40, 50}
average := utils.Math.Average(numbers)           // 30.0
median := utils.Math.Median(numbers)           // 30.0
stdDev := utils.Math.StandardDeviation(numbers) // ~15.81
```

### Currency and Financial Operations
```go
// Currency formatting
amount := 50000.00
formatted := utils.Number.FormatCurrency(amount, "USD")    // "USD 50,000.00"
formattedEUR := utils.Number.FormatCurrency(amount, "EUR")   // "EUR 50,000.00"

// Amount validation
isValidAmount := utils.Number.IsValidAmount(1000)      // true
isValidAmount = utils.Number.IsValidAmount(10000000)   // false (too high)

// Basic fee calculation
fee := utils.Number.CalculateServiceFee(amount)       // Calculated service fee
totalAmount := utils.Math.SafeAdd(amount, fee)         // 50500.00

// Balance operations
currentBalance := 100000.00
newBalance := utils.Math.SafeSubtract(currentBalance, amount)
isSufficient := utils.Number.IsSufficientBalance(currentBalance, amount)
```

## Time Utilities

### Time Zone Operations
```go
import "time"

// Time zone handling
now := time.Now()
localTime := utils.Time.InLocation(now, "Africa/Kampala")
currentTime := utils.Time.NowInLocation("Africa/Kampala")

// Business hours and holidays
isBusinessDay := utils.Time.IsBusinessDay(localTime)            // Is it a weekday?
isWorkingHours := utils.Time.IsBusinessHours(localTime, "Africa/Kampala") // 9AM-5PM?

// Time range utilities
todayStart, todayEnd := utils.Time.Today()                  // Start and end of today
weekStart, weekEnd := utils.Time.ThisWeek("Africa/Kampala") // Current week boundaries
monthStart, monthEnd := utils.Time.ThisMonth("Africa/Kampala") // Current month boundaries
```

### Date Formatting and Operations
```go
// Local time formatting
formatted := utils.Time.FormatLocal(localTime)        // "25-Dec-2024"
timeOnly := utils.Time.FormatLocalTime(localTime)   // "03:30 PM"
dateOnly := utils.Time.FormatLocalDate(localTime)   // "25/12/2024"

// Working days calculation
deadline := utils.Time.AddWorkingDays(localTime, 5)      // 5 business days from now
workingDays := utils.Time.WorkingDaysBetween(start, end) // Count working days

// Business time validation
canProcess := utils.Time.CanProcessTransaction(localTime) // Within banking hours?
nextBusinessDay := utils.Time.NextBusinessDay(localTime)  // Next working day
```

## Validation Utilities

### Struct Validation
```go
type Customer struct {
    Name        string `json:"name" validate:"required,min=2,max=100"`
    PhoneNumber string `json:"phone_number" validate:"required,phone_number"`
    Email       string `json:"email" validate:"required,email"`
    Location    string `json:"location" validate:"required,min=2"`
}

// Automatic validation
func CreateCustomer(customer Customer) error {
    if err := utils.Validate.Struct(customer); err != nil {
        return errors.ValidationFailed[Customer](err)
    }
    return customerService.Create(customer)
}

// Custom validation rules
utils.Validate.RegisterValidation("phone_number", func(fl validator.FieldLevel) bool {
    phone := fl.Field().String()
    return utils.String.IsValidPhoneNumber(phone)
})
```

### Field-Level Validation
```go
// Individual field validation
email := "user@example.com"
isValidEmail := utils.Validate.Var(email, "required,email") // nil if valid

phone := "+256772123456"
isValidPhone := utils.Validate.Var(phone, "phone_number") // nil if valid

// Complex validation scenarios
registrationDate := "2024-01-15"
isValidDate := utils.Validate.Var(registrationDate, "required,datetime=2006-01-02")
```

## UUID and Identifier Generation

### UUID Operations
```go
// UUID generation and validation
customerID := utils.UUID.New()                      // Generate new UUID
transactionID := utils.UUID.NewV4()                 // Generate v4 UUID
sessionID := utils.UUID.NewV7()                    // Generate v7 UUID (time-based)

// UUID validation and parsing
if utils.UUID.IsValid(customerID) {
    // Process valid UUID
}

parsedUUID, err := utils.UUID.Parse("550e8400-e29b-41d4-a716-446655440000")
if err == nil {
    fmt.Printf("Parsed UUID: %s\n", parsedUUID)
}
```

### Business Identifier Generation
```go
// Reference generation
customerRef := utils.String.GenerateReference("CUST")    // "CUST-2024-123456"
invoiceRef := utils.String.GenerateReference("INV")     // "INV-2024-123456"
transactionRef := utils.String.GenerateReference("TXN")  // "TXN-2024-123456"

// Check digit generation for barcodes
barcode := "123456789"
checkDigit := utils.String.GenerateCheckDigit(barcode)  // Mod-10 check digit
ean13 := utils.String.GenerateEAN13(barcode + checkDigit)
```

## Performance Considerations

### Memory-Efficient Operations
```go
// String operations with minimal allocations
largeText := strings.Repeat("sample text ", 10000)
cleaned := utils.String.Clean(largeText)           // Minimal memory allocation
words := utils.String.SplitWords(largeText)        // Efficient splitting

// Numeric operations with overflow protection
largeNumber := math.MaxInt64
result := utils.Math.SafeAdd(largeNumber, 1)       // Graceful overflow handling
safeTotal := utils.Math.SafeSum(numbers...)         // Safe summation

// Time operations with caching
cachedNow := utils.Time.NowCached()                 // Cached current time
formatted := utils.Time.FormatCached(cachedNow)    // Cached formatting
```

## Practical Examples

### Customer Registration
```go
func RegisterCustomer(name, phone, email, location string) (*Customer, error) {
    // Validate phone number
    if !utils.String.IsValidPhoneNumber(phone) {
        return nil, errors.BadRequest("invalid phone number")
    }

    // Validate email
    if !utils.String.IsEmail(email) {
        return nil, errors.BadRequest("invalid email address")
    }

    // Clean and format name
    cleanName := utils.String.ToProperCase(utils.String.Clean(name))

    // Generate customer ID and reference
    customerID := utils.UUID.New()
    customerRef := utils.String.GenerateReference("CUST")

    // Create customer with proper formatting
    now := utils.Time.NowInLocation("Africa/Kampala")
    customer := &Customer{
        ID:          customerID,
        Name:        cleanName,
        PhoneNumber: utils.String.FormatPhoneNumber(phone),
        Email:       email,
        Location:    utils.String.ToTitleCase(location),
        CreatedAt:   now.Format("2006-01-02 15:04:05"),
    }

    return customer, nil
}
```

### Order Processing
```go
func ProcessOrder(phoneNumber string, amount int64) error {
    // Validate phone number
    if !utils.String.IsValidPhoneNumber(phoneNumber) {
        return errors.BadRequest("invalid phone number")
    }

    // Validate amount
    if !utils.Number.IsValidAmount(amount) {
        return errors.BadRequest("invalid order amount")
    }

    // Calculate service fee
    fee := utils.Number.CalculateServiceFee(amount)
    totalAmount := utils.Math.SafeAdd(amount, fee)

    // Generate order reference
    reference := utils.String.GenerateReference("ORD")

    // Process order...
    fmt.Printf("Processing order %s for %s: %d (fee: %d)",
        reference, phoneNumber, amount, fee)

    return nil
}
```

## Available Features

### Currently Implemented ✅
- **String Utilities**: Phone validation, email validation, text formatting, reference generation
- **Numeric Operations**: Safe mathematics, currency formatting, percentage calculations
- **Time Utilities**: Time zone handling, business hours, date formatting
- **Validation**: Struct validation with custom rules, field-level validation
- **UUID Generation**: Multiple UUID versions, business identifiers, check digits
- **Performance**: Memory-efficient operations, overflow protection

### Integration with GoStart

The utils package integrates seamlessly with other GoStart packages:

```go
import (
    "github.com/kod2ulz/gostart/utils"
    "github.com/kod2ulz/gostart/errors"
    "github.com/kod2ulz/gostart/config"
)

func setupApplication() {
    // Use configuration for validation rules
    minLength := config.Get("validation.name_min_length", 2).Int()

    // Use with error handling
    if err := validateInput(input); err != nil {
        return errors.ValidationFailed[InputType](err)
    }
}
```

## Best Practices

### Input Validation
```go
// Always validate external input
func ProcessUserInput(input UserInput) error {
    // Validate email
    if !utils.String.IsEmail(input.Email) {
        return errors.BadRequest("invalid email address")
    }

    // Validate phone number
    if !utils.String.IsValidPhoneNumber(input.Phone) {
        return errors.BadRequest("invalid phone number")
    }

    // Validate amount
    if !utils.Number.IsValidAmount(input.Amount) {
        return errors.BadRequest("invalid amount")
    }

    return nil
}
```

### Safe Mathematical Operations
```go
// Always use safe math for financial calculations
func CalculateTotal(items []Item) (float64, error) {
    var total float64
    for _, item := range items {
        total = utils.Math.SafeAdd(total, item.Price)
    }
    return total, nil
}
```

### Performance Optimization
```go
// Use cached time operations in loops
func ProcessBatch(items []Item) {
    cachedNow := utils.Time.NowCached()

    for _, item := range items {
        item.ProcessedAt = cachedNow
        // Process item...
    }
}
```

---

This package provides essential utilities for Go applications with an emphasis on reliability, performance, and ease of use.