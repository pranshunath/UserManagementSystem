package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/response"
)

// AuthMiddleware verifies the JWT bearer token in the Authorization header.
func AuthMiddleware(tokenManager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header is missing")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid Authorization header format. Expected 'Bearer <token>'")
			return
		}

		tokenStr := parts[1]
		claims, err := tokenManager.VerifyToken(tokenStr)
		if err != nil {
			response.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			return
		}

		// Store verified user claims in Gin context for downstream handlers and RBAC checks
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Set("claims", claims)

		c.Next()
	}
}

// RequireRole enforces Role-Based Access Control (RBAC).
// The user must hold at least one of the allowed roles to proceed.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			response.AbortError(c, http.StatusForbidden, "FORBIDDEN", "No roles associated with request context")
			return
		}

		userRoles, ok := rolesVal.([]string)
		if !ok {
			response.AbortError(c, http.StatusForbidden, "FORBIDDEN", "Invalid roles context format")
			return
		}

		// Check if user has any of the allowed roles
		hasRole := false
		for _, userRole := range userRoles {
			// Super-admin role bypasses restrictions
			if strings.EqualFold(userRole, "Admin") {
				hasRole = true
				break
			}
			for _, allowed := range allowedRoles {
				if strings.EqualFold(userRole, allowed) {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			response.AbortError(c, http.StatusForbidden, "FORBIDDEN", "Access denied: insufficient permissions for this operation")
			return
		}

		c.Next()
	}
}

