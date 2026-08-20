package users

import (
	"errors"
	"log/slog"
	"net/http"

	"ecom-api/internal/auth"
	"ecom-api/internal/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateUser(c *gin.Context) {
	var param CreateUserParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format data tidak valid")
		return
	}

	user, err := h.service.CreateUser(c.Request.Context(), param)
	if err != nil {
		slog.Error("Gagal membuat user", slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal mendaftarkan pengguna")
		return
	}

	response.Success(c, http.StatusCreated, user, nil)
}

func (h *Handler) LoginUser(c *gin.Context) {
	var param LoginUserParam
	if err := c.ShouldBindJSON(&param); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format email/password tidak valid")
		return
	}

	res, err := h.service.LoginUser(c.Request.Context(), param)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Email atau password salah")
			return
		}
		slog.Error("Gagal login user", slog.String("error", err.Error()), slog.String("email", param.Email))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Terjadi kesalahan pada server")
		return
	}

	response.Success(c, http.StatusOK, res, nil)
}

func (h *Handler) GetProfile(c *gin.Context) {
	loggedInUserID, ok := auth.GetUserIDHelper(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Sesi tidak valid")
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), loggedInUserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.Error(c, http.StatusNotFound, response.ErrNotFound, "Profil tidak ditemukan")
			return
		}
		slog.Error("Gagal mengambil profil", slog.Int64("user_id", loggedInUserID), slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal mengambil data profil")
		return
	}

	response.Success(c, http.StatusOK, profile, nil)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	loggedInUserID, ok := auth.GetUserIDHelper(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Sesi tidak valid")
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format data tidak valid")
		return
	}

	updatedProfile, err := h.service.UpdateProfile(c.Request.Context(), loggedInUserID, req)
	if err != nil {
		slog.Error("Gagal update profil", slog.Int64("user_id", loggedInUserID), slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal memperbarui profil")
		return
	}

	response.Success(c, http.StatusOK, updatedProfile, nil)
}
