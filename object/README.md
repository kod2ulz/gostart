# Object Package

The `object` package provides enhanced primitive types with extended functionality for GoStart applications. It offers type-safe wrappers around basic Go types with additional utility methods and fluent APIs.

## Overview

This package enhances Go's primitive types by providing:

- **Extended String Operations**: Additional methods for string manipulation and formatting
- **Generic Number Conversion**: Type-safe numeric parsing and conversion utilities
- **Fluent APIs**: Chain methods for readable data transformations
- **Collections Integration**: Seamless integration with the `collections` package
- **Null-Safe Operations**: Safe operations that handle nil/empty values gracefully

## String Extensions

### Basic String Operations

```go
import "github.com/kod2ulz/gostart/object"

// Create extended string from regular string
str := object.String("hello world")

// Convert back to regular string
regularStr := str.String() // "hello world"

// String splitting with collections integration
parts := str.Split(" ") // collections.List[string]{"hello", "world"}

// Split with multiple separators
csv := object.String("apple,banana;cherry|date")
fruits := csv.SplitMultiple([]string{",", ";", "|"})
// Result: collections.List[string]{"apple", "banana", "cherry", "date"}
```

### String Variations and Formatting

```go
// Generate multiple formatted versions of a string
template := object.String("Hello, {name}!")
variations := template.Variations(
    "Hello, {name}!",
    "Hi, {name}!",
    "Greetings, {name}!",
)
// Result: collections.List[string] with all variations

// Template substitution
greeting := object.String("Hello, {name}!").Substitute("name", "John")
// Result: "Hello, John!"

// Format strings safely
formatted := object.String("User {0} has {1} messages").Format("John", 5)
// Result: "User John has 5 messages"
```

### String Transformations

```go
str := object.String("  Hello World  ")

// Trimming operations
trimmed := str.Trim()          // "Hello World"
trimmedLeft := str.TrimLeft()  // "Hello World  "
trimmedRight := str.TrimRight() // "  Hello World"

// Case operations
upper := str.ToUpper() // "  HELLO WORLD  "
lower := str.ToLower() // "  hello world  "
title := str.ToTitle() // "  Hello World  "

// Padding operations
leftPadded := str.PadLeft(20, "*")     // "*******  Hello World  "
rightPadded := str.PadRight(20, "*")   // "  Hello World  *******"
centerPadded := str.PadCenter(20, "*") // "***  Hello World  ****"
```

### String Validation and Queries

```go
str := object.String("test@example.com")

// Validation operations
isValidEmail := str.IsEmail()            // true
isNumeric := str.IsNumeric()            // false
isEmpty := str.IsEmpty()                // false
isAlpha := str.IsAlpha()                // false (contains @ and .)
isAlphaNumeric := str.IsAlphaNumeric() // false

// Contains operations
containsAt := str.Contains("@")         // true
hasSuffix := str.HasSuffix(".com")     // true
hasPrefix := str.HasPrefix("test")      // true

// Length operations
length := str.Length()                  // 16
isTooLong := str.MaxLength(10)          // false
isTooShort := str.MinLength(5)         // false
```

### String Manipulation

```go
str := object.String("Hello, World!")

// Substring operations
substr := str.Substring(0, 5)           // "Hello"
substrTo := str.SubstringTo(5)          // "Hello"
substrFrom := str.SubstringFrom(7)      // "World!"

// Replacement operations
replaced := str.Replace("World", "Go")  // "Hello, Go!"
replaceAll := str.ReplaceAll("l", "L") // "HeLLo, WorLd!"

// Removal operations
noComma := str.Remove(",")             // "Hello World!"
noHello := str.Remove("Hello")         // ", World!"

// Insertion and concatenation
inserted := str.Insert(7, "Beautiful ") // "Hello, Beautiful World!"
appended := str.Append(" How are you?") // "Hello, World! How are you?"
prepended := str.Prepend("Greeting: ") // "Greeting: Hello, World!"
```

## Number Extensions

### Generic Number Parsing

```go
import "github.com/kod2ulz/gostart/object"

// Parse string to various numeric types
str := "123"

intVal := object.Num[int](str)      // 123 (int)
int8Val := object.Num[int8](str)    // 123 (int8)
int16Val := object.Num[int16](str)  // 123 (int16)
int32Val := object.Num[int32](str)  // 123 (int32)
int64Val := object.Num[int64](str)  // 123 (int64)

uintVal := object.Num[uint](str)    // 123 (uint)
uint8Val := object.Num[uint8](str)  // 123 (uint8)
uint16Val := object.Num[uint16](str) // 123 (uint16)
uint32Val := object.Num[uint32](str) // 123 (uint32)
uint64Val := object.Num[uint64](str) // 123 (uint64)
```

### Safe Number Conversion

```go
// Safe conversion with default values
str := "invalid"

intVal := object.NumSafe[int](str, 0)           // 0 (default)
intVal2 := object.NumSafe[int]("123", 0)         // 123

floatVal := object.NumSafe[float64]("12.34", 0.0) // 12.34
floatVal2 := object.NumSafe[float64]("abc", 0.0)  // 0.0
```

### Number Range Operations

```go
num := object.Num[int]("42")

// Range checking
isInRange := num.Between(0, 100)    // true
isPositive := num.IsPositive()       // true
isNegative := num.IsNegative()       // false
isZero := num.IsZero()              // false

// Clamping operations
clamped := num.Clamp(0, 100)        // 42 (already in range)
clamped2 := num.Clamp(50, 100)       // 50 (clamped to min)
clamped3 := num.Clamp(0, 40)        // 40 (clamped to max)
```

### Number Formatting

```go
num := object.Num[int]("12345")

// Formatting operations
formatted := num.Format()                    // "12345"
withCommas := num.FormatWithCommas()        // "12,345"
asCurrency := num.FormatAsCurrency("USD")    // "$12,345.00"
asPercentage := num.FormatAsPercentage()     // "1,234,500%"

// Scientific notation
scientific := num.FormatScientific(2)        // "1.23e+04"

// Padding operations
padded := num.PadLeft(10, "0")               // "0000012345"
paddedRight := num.PadRight(10, "0")         // "1234500000"
```

## Advanced String Operations

### Pattern Matching

```go
str := object.String("The quick brown fox jumps over the lazy dog")

// Pattern matching
matches := str.Matches(`quick.*fox`)              // true
finds := str.FindAll(`[A-Z][a-z]+`)               // ["The", "Quick", "Brown", "Fox", "Jumps", "Lazy", "Dog"]

// Extract using regular expressions
emails := object.String("Contact us at info@example.com or support@test.org")
extracted := emails.ExtractEmails()               // ["info@example.com", "support@test.org"]

urls := object.String("Visit https://example.com and http://test.org")
extractedUrls := urls.ExtractURLs()                 // ["https://example.com", "http://test.org"]
```

### Text Processing

```go
str := object.String("Hello World! This is a test string.")

// Sentence operations
sentences := str.SplitIntoSentences()              // ["Hello World!", "This is a test string."]
words := str.SplitIntoWords()                      // ["Hello", "World!", "This", "is", "a", "test", "string."]

// Text summarization
summary := str.Summarize(50)                       // First 50 characters
wordSummary := str.SummarizeWords(5)               // First 5 words

// Text statistics
wordCount := str.WordCount()                       // 8
charCount := str.CharCount()                       // 35 (excluding spaces)
readabilityScore := str.ReadabilityScore()         // Flesch reading ease score
```

### String Collections

```go
// Working with multiple strings
strings := object.StringList([]string{"apple", "banana", "cherry"})

// Filtering operations
longWords := strings.Filter(func(s string) bool {
    return len(s) > 5
}) // ["banana", "cherry"]

// Mapping operations
uppercased := strings.Map(func(s string) string {
    return strings.ToUpper(s)
}) // ["APPLE", "BANANA", "CHERRY"]

// Reduction operations
totalLength := strings.Reduce(func(accum int, s string) int {
    return accum + len(s)
}, 0) // 16
```

## Integration with GoStart Framework

### Usage in API Handlers

```go
func UserHandler(ctx contracts.RequestContext) (User, ierrors.Error) {
    var req UserRequest
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return User{}, err
    }

    // Validate email using object package
    emailObj := object.String(req.Email)
    if !emailObj.IsEmail() {
        return User{}, errors.ValidationFailed[UserRequest](
            fmt.Errorf("invalid email format"))
    }

    // Normalize phone number
    phoneObj := object.String(req.Phone)
    normalizedPhone := phoneObj.
        RemoveAll(" ", "-", "(", ")", "+").
        ToLower()

    // Process user data
    user := User{
        Email: emailObj.ToLower().String(),
        Phone: normalizedPhone,
    }

    return userService.CreateUser(user)
}
```

### Usage in Services

```go
type UserService struct {
    repo UserRepository
}

func (s *UserService) ProcessUserData(data string) error {
    dataObj := object.String(data)

    // Validate and clean input
    if dataObj.IsEmpty() {
        return errors.ValidationFailed[string](fmt.Errorf("empty input"))
    }

    cleaned := dataObj.
        Trim().
        ToLower().
        RemoveSpecialCharacters()

    // Extract meaningful information
    if cleaned.IsEmail() {
        return s.processEmail(cleaned.String())
    } else if cleaned.IsNumeric() {
        return s.processNumericData(cleaned.String())
    }

    return s.processTextData(cleaned.String())
}
```

### Configuration and Environment Variables

```go
func LoadConfiguration() (*Config, error) {
    // Parse environment variables with type safety
    port := object.NumSafe[int](os.Getenv("PORT"), "8080")
    timeout := object.NumSafe[int](os.Getenv("TIMEOUT"), "30")

    debugMode := object.String(os.Getenv("DEBUG")).
        ToLower().
        IsOneOf("true", "1", "yes", "on")

    return &Config{
        Port:      port,
        Timeout:   timeout * time.Second,
        DebugMode: debugMode,
    }, nil
}
```

## Performance Considerations

### Memory Efficiency

```go
// Efficient string operations
func processStrings(input []string) []string {
    result := make([]string, 0, len(input))

    for _, s := range input {
        strObj := object.String(s)

        // Chain operations to minimize allocations
        processed := strObj.
            Trim().
            ToLower().
            RemoveSpecialCharacters()

        result = append(result, processed.String())
    }

    return result
}
```

### Batch Operations

```go
// Process multiple numbers efficiently
func processNumbers(inputs []string) ([]int, error) {
    results := make([]int, 0, len(inputs))

    for _, input := range inputs {
        num := object.NumSafe[int](input, 0)
        if num == 0 && !object.String(input).IsNumeric() {
            return nil, fmt.Errorf("invalid number: %s", input)
        }
        results = append(results, num)
    }

    return results, nil
}
```

The object package provides a convenient and type-safe way to work with primitive types in GoStart applications, offering extended functionality while maintaining performance and integration with the broader framework.