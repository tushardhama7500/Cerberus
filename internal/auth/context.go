package auth

import "context"

// contextKey is an unexported type to prevent key collisions in context.
// Using a plain string like "user" is a common beginner mistake that causes
// subtle bugs when multiple packages store things in the same context.
type contextKey string

const userClaimsKey contextKey = "userClaims"

// InjectClaims stores auth claims in the request context.
// Called by the JWT middleware before the request reaches the resolver.
func InjectClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, userClaimsKey, claims)
}

// ExtractClaims retrieves auth claims from context.
// Returns nil if not authenticated — resolvers check this.
func ExtractClaims(ctx context.Context) *Claims {
	claims, _ := ctx.Value(userClaimsKey).(*Claims)
	return claims
}
