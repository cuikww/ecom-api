package products

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"ecom-api/internal/response"
	"ecom-api/internal/utils"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) ListProducts(c *gin.Context) {
	pagination := utils.GeneratePaginationFromRequest(c)

	products, err := h.service.ListProducts(c.Request.Context(), pagination)
	if err != nil {
		slog.Error("ListProducts gagal", slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal mengambil daftar produk")
		return
	}

	response.Success(c, http.StatusOK, products, pagination)
}

func (h *Handler) FindProductsByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format ID Produk tidak valid")
		return
	}

	product, err := h.service.FindProductsByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, response.ErrNotFound, "Produk tidak ditemukan")
			return
		}

		slog.Error("FindProductsByID gagal", slog.Int64("product_id", id), slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal mengambil data produk")
		return
	}

	response.Success(c, http.StatusOK, product, nil)
}

func (h *Handler) CreateProduct(c *gin.Context) {
	var params ProductParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format data produk tidak valid")
		return
	}

	product, err := h.service.CreateProduct(c.Request.Context(), params)
	if err != nil {
		slog.Error("CreateProduct gagal", slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal menambahkan produk baru")
		return
	}

	response.Success(c, http.StatusCreated, product, nil)
}

func (h *Handler) UpdateProduct(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format ID Produk tidak valid")
		return
	}

	var params ProductParams
	if err := c.ShouldBindJSON(&params); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format data yang diperbarui tidak valid")
		return
	}

	params.ID = id

	product, err := h.service.UpdateProduct(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			response.Error(c, http.StatusNotFound, response.ErrNotFound, "Produk yang ingin diperbarui tidak ditemukan")
			return
		}

		slog.Error("UpdateProduct gagal", slog.Int64("product_id", id), slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal memperbarui data produk")
		return
	}

	response.Success(c, http.StatusOK, product, nil)
}
