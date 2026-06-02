package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"cerberus/config"
	"cerberus/internal/auth"

	"github.com/gin-gonic/gin"
)

// JWTAuth extracts and validates the Bearer token from the Authorization header.
// It injects the parsed claims into the request context.
// The GraphQL handler runs AFTER this middleware, so resolvers can trust the context.
//
// Note: We do NOT return 401 for missing tokens here — some GraphQL operations
// (register, login) are public. We let the resolver decide if auth is required.
// If a token IS present and invalid, we reject immediately.
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("2. JWT middleware called")
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "malformed authorization header"})
			c.Abort()
			return
		}

		claims, err := auth.ValidateToken(parts[1], cfg.JWT.Secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		// Inject into request context — GraphQL handler will carry this context
		ctx := auth.InjectClaims(c.Request.Context(), claims)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
