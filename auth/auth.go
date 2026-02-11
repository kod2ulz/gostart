package auth

// Auth provides authentication and authorization utilities for gostart-based applications.
//
// This package provides:
//   - Minimal User and SessionUser interfaces that can be extended in your project
//   - Helper functions (GetUser, GetSessionUser) to retrieve users from context
//   - GinContext adapter for framework-agnostic middleware
//   - Example patterns for implementing authentication middleware
//
// The heavy lifting (token loading, validation, context storage) is handled here,
// but the actual user model and authorization logic (roles, permissions) are defined
// in your project to keep the framework flexible.
//
// Example usage in your project:
//
//	// Define your user type
//	type StaffUser struct {
//	    ID   uuid.UUID
//	    Name string
//	    Roles []string
//	    auth.User // Embed for interface compliance
//	}
//
//	// Create a token verifier
//	type TokenVerifier struct {
//	    service *TokenService
//	}
//
//	func (v *TokenVerifier) VerifyToken(ctx contracts.RequestContext, req TokenRequest) (StaffUser, error) {
//	    // Your verification logic here
//	    return user, nil
//	}
//
//	// Use in middleware
//	auther := auth.WithUser(&TokenVerifier{service: tokenService})
//	router.Group("/api", auther).GET("/users", handler)
