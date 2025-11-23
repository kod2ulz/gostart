package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/ierrors"
	"github.com/kod2ulz/gostart/errors"
	"github.com/o1egl/paseto"
)

// TokenProvider interface for generating and validating tokens
type TokenProvider interface {
	Generate(userID uuid.UUID, claims map[string]interface{}, duration time.Duration) (string, error)
	Validate(token string) (*TokenClaims, error)
	Refresh(token string, duration time.Duration) (string, error)
}

// TokenClaims represents the claims in a token
type TokenClaims struct {
	UserID    uuid.UUID              `json:"user_id"`
	ExpiresAt time.Time              `json:"exp"`
	IssuedAt  time.Time              `json:"iat"`
	Issuer    string                 `json:"iss"`
	Custom    map[string]interface{} `json:"custom,omitempty"`
}

// JWTProvider implements JWT token generation and validation
type JWTProvider struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	algorithm  string
}

// NewJWTProvider creates a new JWT token provider
func NewJWTProvider(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, issuer string) *JWTProvider {
	return &JWTProvider{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
		algorithm:  "RS256",
	}
}

// NewJWTProviderHS256 creates a new JWT token provider with HMAC-SHA256
func NewJWTProviderHS256(secret []byte, issuer string) (*JWTProvider, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("secret must be at least 32 bytes")
	}
	return &JWTProvider{
		privateKey: nil, // Not used for HMAC
		publicKey:  nil, // Not used for HMAC
		issuer:     issuer,
		algorithm:  "HS256",
	}, nil
}

// Generate generates a JWT token
func (j *JWTProvider) Generate(userID uuid.UUID, claims map[string]interface{}, duration time.Duration) (string, error) {
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     now.Add(duration).Unix(),
		"iat":     now.Unix(),
		"iss":     j.issuer,
	}

	// Add custom claims
	for k, v := range claims {
		jwtClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)

	if j.algorithm == "HS256" {
		// For HMAC, we would use the secret from configuration
		return "", fmt.Errorf("HS256 signing not fully implemented - use RS256")
	}

	return token.SignedString(j.privateKey)
}

// Validate validates a JWT token
func (j *JWTProvider) Validate(tokenString string) (*TokenClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if j.algorithm == "RS256" {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return j.publicKey, nil
		}
		return nil, fmt.Errorf("unsupported algorithm: %s", j.algorithm)
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims format")
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid user_id claim")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid exp claim")
	}

	iat, ok := claims["iat"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid iat claim")
	}

	issuer, _ := claims["iss"].(string)

	// Extract custom claims
	customClaims := make(map[string]interface{})
	for k, v := range claims {
		if k != "user_id" && k != "exp" && k != "iat" && k != "iss" {
			customClaims[k] = v
		}
	}

	return &TokenClaims{
		UserID:    userID,
		ExpiresAt: time.Unix(int64(exp), 0),
		IssuedAt:  time.Unix(int64(iat), 0),
		Issuer:    issuer,
		Custom:    customClaims,
	}, nil
}

// Refresh refreshes a JWT token
func (j *JWTProvider) Refresh(tokenString string, duration time.Duration) (string, error) {
	claims, err := j.Validate(tokenString)
	if err != nil {
		return "", err
	}

	return j.Generate(claims.UserID, claims.Custom, duration)
}

// PASETOProvider implements PASETO token generation and validation
type PASETOProvider struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	issuer     string
}

// NewPASETOProvider creates a new PASETO token provider
func NewPASETOProvider(issuer string) (*PASETOProvider, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PASETO keys: %w", err)
	}

	return &PASETOProvider{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
	}, nil
}

// NewPASETOProviderWithKeys creates a new PASETO provider with existing keys
func NewPASETOProviderWithKeys(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, issuer string) *PASETOProvider {
	return &PASETOProvider{
		privateKey: privateKey,
		publicKey:  publicKey,
		issuer:     issuer,
	}
}

// Generate generates a PASETO token
func (p *PASETOProvider) Generate(userID uuid.UUID, claims map[string]interface{}, duration time.Duration) (string, error) {
	now := time.Now()

	jsonToken := paseto.JSONToken{
		Audience:   p.issuer,
		Issuer:     p.issuer,
		Jti:        uuid.New().String(),
		Subject:    userID.String(),
		IssuedAt:   now,
		Expiration: now.Add(duration),
	}

	// Add custom claims
	for k, v := range claims {
		jsonToken.Set(k, v)
	}

	v2 := paseto.NewV2()
	token, err := v2.Sign(p.privateKey, jsonToken, nil)
	if err != nil {
		return "", fmt.Errorf("failed to sign PASETO token: %w", err)
	}

	return token, nil
}

// Validate validates a PASETO token
func (p *PASETOProvider) Validate(tokenString string) (*TokenClaims, error) {
	v2 := paseto.NewV2()
	var jsonToken paseto.JSONToken
	var footer string

	err := v2.Verify(tokenString, p.publicKey, &jsonToken, &footer)
	if err != nil {
		return nil, fmt.Errorf("failed to verify PASETO token: %w", err)
	}

	userID, err := uuid.Parse(jsonToken.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid subject (user_id): %w", err)
	}

	// Extract custom claims
	customClaims := make(map[string]interface{})
	for k, v := range jsonToken.Get("custom").(map[string]interface{}) {
		customClaims[k] = v
	}

	return &TokenClaims{
		UserID:    userID,
		ExpiresAt: jsonToken.Expiration,
		IssuedAt:  jsonToken.IssuedAt,
		Issuer:    jsonToken.Issuer,
		Custom:    customClaims,
	}, nil
}

// Refresh refreshes a PASETO token
func (p *PASETOProvider) Refresh(tokenString string, duration time.Duration) (string, error) {
	claims, err := p.Validate(tokenString)
	if err != nil {
		return "", err
	}

	return p.Generate(claims.UserID, claims.Custom, duration)
}

// TokenService manages token operations
type TokenService struct {
	provider         TokenProvider
	accessDuration   time.Duration
	refreshDuration  time.Duration
}

// NewTokenService creates a new token service
func NewTokenService(provider TokenProvider, accessDuration, refreshDuration time.Duration) *TokenService {
	if accessDuration == 0 {
		accessDuration = 15 * time.Minute
	}
	if refreshDuration == 0 {
		refreshDuration = 7 * 24 * time.Hour
	}

	return &TokenService{
		provider:         provider,
		accessDuration:   accessDuration,
		refreshDuration:  refreshDuration,
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (t *TokenService) GenerateTokenPair(userID uuid.UUID, claims map[string]interface{}) (*TokenPair, ierrors.Error) {
	accessToken, err := t.provider.Generate(userID, claims, t.accessDuration)
	if err != nil {
		return nil, errors.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := t.provider.Generate(userID, claims, t.refreshDuration)
	if err != nil {
		return nil, errors.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(t.accessDuration.Seconds()),
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken validates an access token
func (t *TokenService) ValidateAccessToken(token string) (*TokenClaims, ierrors.Error) {
	claims, err := t.provider.Validate(token)
	if err != nil {
		return nil, errors.ServiceUnauthorised(err)
	}

	if time.Now().After(claims.ExpiresAt) {
		return nil, errors.Errorf("token expired")
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a refresh token
func (t *TokenService) RefreshAccessToken(refreshToken string) (*TokenPair, ierrors.Error) {
	claims, err := t.provider.Validate(refreshToken)
	if err != nil {
		return nil, errors.ServiceUnauthorised(err)
	}

	if time.Now().After(claims.ExpiresAt) {
		return nil, errors.Errorf("refresh token expired")
	}

	return t.GenerateTokenPair(claims.UserID, claims.Custom)
}

// TokenPair represents a pair of access and refresh tokens
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}
