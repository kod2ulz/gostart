# Request Processing and Parameter Loading

This guide covers advanced request processing patterns and parameter loading techniques in GoStart.

## Request Processing Architecture

### Multi-Source Parameter Loading

The API package automatically loads request parameters from multiple sources:

```go
type UserProfileRequest struct {
    UserID    string `json:"user_id" param:"id" validate:"required"`
    Fields    string `query:"fields" default:"name,email"`
    Version   string `header:"X-API-Version"`
    Include   string `query:"include"`
    Format    string `query:"format" default:"json"`
}

func GetUserProfileHandler(ctx contracts.RequestContext) (UserProfileResponse, ierrors.Error) {
    var req UserProfileRequest

    // Automatically loads from:
    // - JSON body: user_id
    // - URL params: id (maps to user_id)
    // - Query params: fields, include, format
    // - Headers: X-API-Version
    if err := contracts.ContextLoad(ctx.Context(), &req); err != nil {
        return UserProfileResponse{}, err
    }

    // Process request
    profile, err := userService.GetProfile(req.UserID, req.Fields)
    if err != nil {
        return UserProfileResponse{}, err
    }

    return UserProfileResponse{
        Profile: profile,
        Meta: ResponseMeta{
            Fields:  req.Fields,
            Include: req.Include,
            Format:  req.Format,
        },
    }, nil
}
```

### Request Parameter Interface

```go
type RequestParam interface {
    // Load parameters from request
    RequestLoad(ctx contracts.RequestContext) (RequestParam, error)

    // Validate loaded parameters
    Validate(ctx contracts.RequestContext) error

    // Get context key for storage
    ContextKey() string

    // Get metadata context key
    MetadataContextKey() string

    // Get references context key
    ReferencesContextKey() string

    // Set response metadata
    SetResponseMetadata(ctx contracts.RequestContext, meta *Metadata) error

    // Set response references
    SetResponseReference(ctx contracts.RequestContext, key string, value interface{}) error

    // Load from context
    ContextLoad(ctx context.Context) (RequestParam, error)

    // Load parameters from context into out parameter
    LoadFromContext(ctx context.Context, out RequestParam) error
}
```

## Advanced Parameter Loading

### Custom Parameter Loading

```go
type SearchRequest struct {
    Query      string            `json:"query" query:"q" validate:"required"`
    Filters    map[string]string `json:"filters" query:"filter"`
    Sort       string            `json:"sort" query:"sort" default:"relevance"`
    Pagination PaginationInfo    `json:"pagination"`
    Aggregations []Aggregation   `json:"aggregations"`
}

func (r *SearchRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    // Load query from multiple sources
    if query := ctx.Query("q"); query != "" {
        r.Query = query
    } else if query := ctx.FormValue("query"); query != "" {
        r.Query = query
    }

    // Load filters from query parameters
    filters := make(map[string]string)
    for key, values := range ctx.Request().URL.Query() {
        if strings.HasPrefix(key, "filter[") && strings.HasSuffix(key, "]") {
            field := strings.TrimPrefix(strings.TrimSuffix(key, "]"), "filter[")
            if len(values) > 0 {
                filters[field] = values[0]
            }
        }
    }
    r.Filters = filters

    // Load sort parameters
    if sort := ctx.Query("sort"); sort != "" {
        r.Sort = sort
    }

    // Load pagination from query or headers
    rPagination := PaginationInfo{
        Page:     ctx.QueryOrDefault("page", "1").Int(),
        PageSize: ctx.QueryOrDefault("page_size", "20").Int(),
    }
    r.Pagination = rPagination

    return r, nil
}

func (r *SearchRequest) Validate(ctx contracts.RequestContext) error {
    if r.Query == "" {
        return errors.New("search query is required")
    }

    if r.Pagination.Page < 1 {
        return errors.New("page must be greater than 0")
    }

    if r.Pagination.PageSize > 100 {
        return errors.New("page size cannot exceed 100")
    }

    return nil
}
```

### Complex Data Type Loading

```go
type FileUploadRequest struct {
    File       multipart.FileHeader `json:"-" validate:"required"`
    Metadata   FileMetadata        `json:"metadata"`
    Processing ProcessingOptions   `json:"processing"`
}

type FileMetadata struct {
    Name        string `json:"name" validate:"required"`
    Type        string `json:"type" validate:"required"`
    Category    string `json:"category"`
    Tags        []string `json:"tags"`
    Description string `json:"description"`
}

type ProcessingOptions struct {
    Resize       ResizeOptions `json:"resize"`
    Quality      int           `json:"quality" validate:"min=1,max=100"`
    Format       string        `json:"format" validate:"oneof=jpeg,png,webp"`
    Watermark    bool          `json:"watermark"`
}

func (r *FileUploadRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    // Load file from multipart form
    file, header, err := ctx.Request().FormFile("file")
    if err != nil {
        return nil, err
    }
    defer file.Close()

    r.File = *header

    // Load metadata from form fields or JSON
    if metadataJSON := ctx.FormValue("metadata"); metadataJSON != "" {
        var metadata FileMetadata
        if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
            return nil, err
        }
        r.Metadata = metadata
    } else {
        // Load metadata from individual form fields
        r.Metadata = FileMetadata{
            Name:        ctx.FormValue("name"),
            Type:        ctx.FormValue("type"),
            Category:    ctx.FormValue("category"),
            Description: ctx.FormValue("description"),
        }

        // Load tags from multiple form fields
        if tags := ctx.FormValue("tags"); tags != "" {
            r.Metadata.Tags = strings.Split(tags, ",")
        }
    }

    // Load processing options
    if processingJSON := ctx.FormValue("processing"); processingJSON != "" {
        var processing ProcessingOptions
        if err := json.Unmarshal([]byte(processingJSON), &processing); err != nil {
            return nil, err
        }
        r.Processing = processing
    }

    return r, nil
}
```

### Nested Object Loading

```go
type OrderRequest struct {
    CustomerID string       `json:"customer_id" validate:"required"`
    Items      []OrderItem  `json:"items" validate:"required,min=1"`
    Shipping   ShippingInfo `json:"shipping"`
    Payment    PaymentInfo  `json:"payment"`
    Metadata   OrderMetadata `json:"metadata"`
}

type OrderItem struct {
    ProductID string  `json:"product_id" validate:"required"`
    Quantity  int     `json:"quantity" validate:"min=1"`
    Price     float64 `json:"price" validate:"min=0"`
    Options   []ItemOption `json:"options"`
}

type ShippingInfo struct {
    Address    Address `json:"address" validate:"required"`
    Method     string  `json:"method" validate:"required"`
    Expedited  bool    `json:"expedited"`
    Tracking   string  `json:"tracking,omitempty"`
}

type Address struct {
    Street     string `json:"street" validate:"required"`
    City       string `json:"city" validate:"required"`
    State      string `json:"state" validate:"required"`
    PostalCode string `json:"postal_code" validate:"required"`
    Country    string `json:"country" validate:"required"`
}

func (r *OrderRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    // Load main order data
    if err := ctx.ShouldBindJSON(r); err != nil {
        return nil, err
    }

    // Load additional data from query parameters
    if customerID := ctx.Query("customer_id"); customerID != "" {
        r.CustomerID = customerID
    }

    // Load metadata from headers
    r.Metadata = OrderMetadata{
        Source:    ctx.Header("X-Order-Source"),
        Reference: ctx.Header("X-Order-Reference"),
        UserAgent: ctx.Header("User-Agent"),
    }

    return r, nil
}

func (r *OrderRequest) Validate(ctx contracts.RequestContext) error {
    // Validate order total
    total := 0.0
    for _, item := range r.Items {
        total += item.Price * float64(item.Quantity)
    }

    if total <= 0 {
        return errors.New("order total must be greater than 0")
    }

    // Validate shipping based on location
    if r.Shipping.Address.Country == "US" && len(r.Shipping.Address.PostalCode) != 5 {
        return errors.New("invalid US postal code")
    }

    // Validate payment method availability
    if r.Payment.Method == "paypal" && r.Shipping.Address.Country != "US" {
        return errors.New("PayPal not available for international orders")
    }

    return nil
}
```

## Validation Patterns

### Multi-Level Validation

```go
type UserRegistrationRequest struct {
    Profile    UserProfile  `json:"profile" validate:"required"`
    Credentials UserCredentials `json:"credentials" validate:"required"`
    Preferences UserPreferences `json:"preferences"`
    Marketing  MarketingPrefs `json:"marketing"`
}

type UserProfile struct {
    FirstName string `json:"first_name" validate:"required"`
    LastName  string `json:"last_name" validate:"required"`
    Email     string `json:"email" validate:"required,email"`
    Phone     string `json:"phone" validate:"e164"`
    BirthDate string `json:"birth_date" validate:"required,datetime=2006-01-02"`
    Address   Address `json:"address" validate:"required"`
}

type UserCredentials struct {
    Password string `json:"password" validate:"required,min=8"`
    Confirm  string `json:"confirm" validate:"required"`
}

func (r *UserRegistrationRequest) Validate(ctx contracts.RequestContext) error {
    // Level 1: Basic field validation (handled by tags)

    // Level 2: Cross-field validation
    if r.Credentials.Password != r.Credentials.Confirm {
        return errors.New("passwords do not match")
    }

    // Level 3: Business logic validation
    if r.Profile.BirthDate != "" {
        birthDate, err := time.Parse("2006-01-02", r.Profile.BirthDate)
        if err != nil {
            return errors.New("invalid birth date format")
        }

        age := time.Since(birthDate).Hours() / 24 / 365
        if age < 13 {
            return errors.New("user must be at least 13 years old")
        }
    }

    // Level 4: External validation
    if emailExists, _ := userService.EmailExists(r.Profile.Email); emailExists {
        return errors.New("email already registered")
    }

    // Level 5: Context-aware validation
    if r.Profile.Address.Country == "US" && r.Profile.Phone == "" {
        return errors.New("phone number is required for US users")
    }

    return nil
}
```

### Conditional Validation

```go
type BookingRequest struct {
    Type      string        `json:"type" validate:"required,oneof=hotel,flight,car"`
    Hotel     *HotelBooking `json:"hotel,omitempty" validate:"required_if=Type hotel"`
    Flight    *FlightBooking `json:"flight,omitempty" validate:"required_if=Type flight"`
    Car       *CarBooking   `json:"car,omitempty" validate:"required_if=Type car"`
    Travelers []Traveler    `json:"travelers" validate:"required,min=1"`
    Payment   PaymentInfo   `json:"payment" validate:"required"`
}

func (r *BookingRequest) Validate(ctx contracts.RequestContext) error {
    // Type-specific validation
    switch r.Type {
    case "hotel":
        if r.Hotel == nil {
            return errors.New("hotel booking details are required")
        }
        if r.Hotel.CheckIn.After(r.Hotel.CheckOut) {
            return errors.New("check-in date must be before check-out date")
        }
        if r.Hotel.Guests > r.Hotel.Room.MaxOccupancy {
            return errors.New("number of guests exceeds room capacity")
        }

    case "flight":
        if r.Flight == nil {
            return errors.New("flight booking details are required")
        }
        if r.Flight.Departure.After(r.Flight.Return) {
            return errors.New("departure date must be before return date")
        }
        if len(r.Flight.Passengers) != len(r.Travelers) {
            return errors.New("number of passengers must match number of travelers")
        }

    case "car":
        if r.Car == nil {
            return errors.New("car booking details are required")
        }
        if r.Car.PickUp.After(r.Car.DropOff) {
            return errors.New("pickup date must be before drop-off date")
        }
        if r.Car.DriverAge < 21 {
            return errors.New("driver must be at least 21 years old")
        }
    }

    // Payment validation
    if r.Payment.Method == "credit_card" {
        if r.Payment.CardNumber == "" {
            return errors.New("credit card number is required")
        }
        if !isValidCreditCard(r.Payment.CardNumber) {
            return errors.New("invalid credit card number")
        }
    }

    return nil
}
```

### Custom Validation Rules

```go
type CustomValidationRequest struct {
    TaxID      string `json:"tax_id" validate:"required"`
    License    string `json:"license" validate:"required"`
    Documents  []Document `json:"documents" validate:"required,min=1"`
}

func (r *CustomValidationRequest) Validate(ctx contracts.RequestContext) error {
    // Custom tax ID validation
    if !isValidTaxID(r.TaxID) {
        return errors.New("invalid tax ID format")
    }

    // Custom license validation
    if !isValidLicense(r.License) {
        return errors.New("invalid license number")
    }

    // Document validation
    for _, doc := range r.Documents {
        if !isValidDocument(doc.Type, doc.Number) {
            return fmt.Errorf("invalid %s document number", doc.Type)
        }
        if doc.ExpiryDate.Before(time.Now()) {
            return fmt.Errorf("%s document has expired", doc.Type)
        }
    }

    return nil
}

func isValidTaxID(taxID string) bool {
    // Example: US Social Security Number validation
    if len(taxID) != 11 {
        return false
    }

    parts := strings.Split(taxID, "-")
    if len(parts) != 3 {
        return false
    }

    if len(parts[0]) != 3 || len(parts[1]) != 2 || len(parts[2]) != 4 {
        return false
    }

    // Check if all parts are numeric
    for _, part := range parts {
        if !strings.ContainsAny(part, "0123456789") {
            return false
        }
    }

    return true
}

func isValidLicense(license string) bool {
    // Example: Driver's license validation
    // This would vary by state/country
    return len(license) >= 6 && len(license) <= 15
}

func isValidDocument(docType, docNumber string) bool {
    switch docType {
    case "passport":
        return len(docNumber) == 9
    case "visa":
        return len(docNumber) == 8
    case "id_card":
        return len(docNumber) >= 8 && len(docNumber) <= 12
    default:
        return false
    }
}
```

## Request Transformation

### Data Normalization

```go
type NormalizedRequest struct {
    Email     string `json:"email"`
    Phone     string `json:"phone"`
    Address   string `json:"address"`
    Amount    float64 `json:"amount"`
    Currency  string `json:"currency"`
}

func (r *NormalizedRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    // Load raw data
    if err := ctx.ShouldBindJSON(r); err != nil {
        return nil, err
    }

    // Normalize email
    r.Email = strings.TrimSpace(strings.ToLower(r.Email))

    // Normalize phone number
    r.Phone = normalizePhoneNumber(r.Phone)

    // Normalize address
    r.Address = normalizeAddress(r.Address)

    // Normalize amount and currency
    r.Amount = math.Round(r.Amount*100) / 100
    r.Currency = strings.ToUpper(r.Currency)

    return r, nil
}

func normalizePhoneNumber(phone string) string {
    // Remove all non-digit characters
    digits := strings.Map(func(r rune) rune {
        if r >= '0' && r <= '9' {
            return r
        }
        return -1
    }, phone)

    // Add country code if missing
    if len(digits) == 10 {
        digits = "1" + digits
    }

    // Format as E.164
    if len(digits) == 11 {
        return "+" + digits[:1] + "-" + digits[1:4] + "-" + digits[4:7] + "-" + digits[7:]
    }

    return phone
}

func normalizeAddress(address string) string {
    // Standardize address format
    parts := strings.Split(address, ",")
    for i, part := range parts {
        parts[i] = strings.TrimSpace(strings.Title(part))
    }
    return strings.Join(parts, ", ")
}
```

### Request Enrichment

```go
type EnrichedRequest struct {
    RawData    map[string]interface{} `json:"raw_data"`
    Enriched   EnrichedData          `json:"enriched"`
    Metadata   RequestMetadata       `json:"metadata"`
    Context    RequestContext        `json:"context"`
}

type EnrichedData struct {
    Location   GeoLocation `json:"location"`
    Device     DeviceInfo  `json:"device"`
    Session    SessionInfo `json:"session"`
    User       UserInfo   `json:"user"`
}

func (r *EnrichedRequest) RequestLoad(ctx contracts.RequestContext) (contracts.RequestParam, error) {
    // Load raw data
    var rawData map[string]interface{}
    if err := ctx.ShouldBindJSON(&rawData); err != nil {
        return nil, err
    }
    r.RawData = rawData

    // Enrich with location data
    if ip := ctx.ClientIP(); ip != "" {
        location, err := geoService.LookupIP(ip)
        if err == nil {
            r.Enriched.Location = location
        }
    }

    // Enrich with device info
    r.Enriched.Device = DeviceInfo{
        UserAgent: ctx.Header("User-Agent"),
        Type:      detectDeviceType(ctx.Header("User-Agent")),
        OS:        detectOS(ctx.Header("User-Agent")),
    }

    // Enrich with session info
    if sessionID := ctx.Header("X-Session-ID"); sessionID != "" {
        session, err := sessionService.Get(sessionID)
        if err == nil {
            r.Enriched.Session = session
        }
    }

    // Enrich with user info
    if userID := ctx.Value("user_id"); userID != nil {
        user, err := userService.GetByID(userID.(string))
        if err == nil {
            r.Enriched.User = user
        }
    }

    // Add request metadata
    r.Metadata = RequestMetadata{
        Timestamp:   time.Now(),
        IP:         ctx.ClientIP(),
        UserAgent:  ctx.Header("User-Agent"),
        Referer:    ctx.Header("Referer"),
        Method:     ctx.Method(),
        Path:       ctx.Path(),
        Query:      ctx.Request().URL.RawQuery,
    }

    return r, nil
}
```

## Error Handling and Validation

### Structured Error Responses

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Code    string `json:"code"`
    Value   interface{} `json:"value,omitempty"`
}

type ValidationErrorResponse struct {
    Success   bool               `json:"success"`
    Error     string             `json:"error"`
    Details   []ValidationError  `json:"details"`
    RequestID string             `json:"request_id"`
}

func handleValidationError(ctx contracts.RequestContext, err error) (interface{}, int) {
    validationErrors := extractValidationErrors(err)

    response := ValidationErrorResponse{
        Success:   false,
        Error:     "Validation failed",
        Details:   validationErrors,
        RequestID: ctx.Header("X-Request-ID"),
    }

    return response, http.StatusBadRequest
}

func extractValidationErrors(err error) []ValidationError {
    var errors []ValidationError

    // Handle validation errors from different sources
    switch v := err.(type) {
    case validator.ValidationErrors:
        for _, fieldErr := range v {
            errors = append(errors, ValidationError{
                Field:   fieldErr.Field(),
                Message: getValidationMessage(fieldErr),
                Code:    getValidationCode(fieldErr),
                Value:   fieldErr.Value(),
            })
        }
    case *json.UnmarshalTypeError:
        errors = append(errors, ValidationError{
            Field:   v.Field,
            Message: fmt.Sprintf("Invalid type: expected %s, got %s", v.Type, v.Value),
            Code:    "INVALID_TYPE",
            Value:   v.Value,
        })
    default:
        errors = append(errors, ValidationError{
            Field:   "general",
            Message: err.Error(),
            Code:    "VALIDATION_ERROR",
        })
    }

    return errors
}

func getValidationMessage(fieldErr validator.FieldError) string {
    switch fieldErr.Tag() {
    case "required":
        return fmt.Sprintf("%s is required", fieldErr.Field())
    case "email":
        return fmt.Sprintf("%s must be a valid email address", fieldErr.Field())
    case "min":
        if fieldErr.Kind() == reflect.String {
            return fmt.Sprintf("%s must be at least %s characters", fieldErr.Field(), fieldErr.Param())
        }
        return fmt.Sprintf("%s must be at least %s", fieldErr.Field(), fieldErr.Param())
    case "max":
        if fieldErr.Kind() == reflect.String {
            return fmt.Sprintf("%s must be at most %s characters", fieldErr.Field(), fieldErr.Param())
        }
        return fmt.Sprintf("%s must be at most %s", fieldErr.Field(), fieldErr.Param())
    default:
        return fmt.Sprintf("%s failed validation: %s", fieldErr.Field(), fieldErr.Tag())
    }
}
```

These request processing patterns provide a comprehensive framework for handling complex API requests in GoStart applications.