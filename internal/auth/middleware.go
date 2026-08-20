package auth

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"ecom-api/internal/response"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(jwtService *JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			slog.Warn("Akses ditolak",
				slog.String("reason", "Header Authorization kosong"),
				slog.String("ip", c.ClientIP()),
				slog.String("path", c.Request.URL.Path),
			)

			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Header authorization wajib diisi")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			slog.Warn("Akses ditolak",
				slog.String("reason", "Format header tidak valid (Bukan Bearer)"),
				slog.String("ip", c.ClientIP()),
			)
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Format authorization header tidak valid")
			return
		}

		claims, err := jwtService.ParseToken(parts[1])
		if err != nil {
			slog.Warn("Akses ditolak",
				slog.String("reason", "Token expired atau signature tidak valid"),
				slog.String("error", err.Error()),
			)
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Token tidak valid atau telah kedaluwarsa")
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			slog.Warn("Akses ditolak",
				slog.String("reason", "ID user di dalam token bukan angka yang valid"),
			)
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Data identitas pada token tidak valid")
			return
		}

		c.Set("userID", userID)
		c.Set("userRole", claims.Role)
		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleRaw, exists := c.Get("userRole")
		if !exists {
			slog.Warn("Akses ditolak", slog.String("reason", "Role tidak ditemukan di context"))
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Sesi tidak valid")
			return
		}
		userRole, ok := userRoleRaw.(string)
		if !ok {
			response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Kesalahan sistem saat menerima otorisasi")
		}
		isAllowed := false

		for _, role := range allowedRoles {
			if userRole == role {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			slog.Warn("Otorisasi gagal",
				slog.String("user_role", userRole),
				slog.String("required_roles", strings.Join(allowedRoles, ",")),
				slog.String("path", c.Request.URL.Path),
			)
			response.Error(c, http.StatusForbidden, "ERR_FORBIDDEN", "Anda tidak memiliki akses")
			return
		}

		c.Next()
	}
}

func GetUserIDHelper(c *gin.Context) (int64, bool) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		return 0, false
	}
	userID, ok := userIDRaw.(int64)
	return userID, ok
}
