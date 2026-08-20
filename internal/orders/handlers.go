package orders

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"ecom-api/internal/auth"
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

func (h *Handler) PlaceOrder(c *gin.Context) {
	loggedInUserID, ok := auth.GetUserIDHelper(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Sesi tidak valid atau telah berakhir")
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format data pesanan tidak valid")
		return
	}

	req.CustomerID = loggedInUserID

	order, err := h.service.PlaceOrder(c.Request.Context(), req)
	if err != nil {
		slog.Error("PlaceOrder gagal", slog.Int64("user_id", loggedInUserID), slog.String("error", err.Error()))

		if errors.Is(err, ErrorProductNotFound) {
			response.Error(c, http.StatusNotFound, response.ErrNotFound, "Beberapa produk tidak ditemukan")
			return
		}

		if errors.Is(err, ErrorProductNoStock) {
			response.Error(c, http.StatusConflict, response.ErrConflict, "Stok produk tidak mencukupi untuk pesanan ini")
			return
		}

		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal membuat pesanan, silakan coba lagi")
		return
	}

	response.Success(c, http.StatusCreated, order, nil)
}

func (h *Handler) ListOrdersByCustomer(c *gin.Context) {
	loggedInUserID, ok := auth.GetUserIDHelper(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.ErrUnauthorized, "Sesi tidak valid atau telah berakhir")
		return
	}

	customerID, err := strconv.ParseInt(c.Param("customer_id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format ID Customer tidak valid")
		return
	}

	// Validasi IDOR
	if customerID != loggedInUserID {
		response.Error(c, http.StatusForbidden, "ERR_FORBIDDEN", "Anda tidak memiliki akses ke data pesanan pengguna lain")
		return
	}

	pagination := utils.GeneratePaginationFromRequest(c)

	orders, err := h.service.ListOrdersByCustomer(c.Request.Context(), customerID, pagination)
	if err != nil {
		slog.Error("ListOrdersByCustomer gagal", slog.Int64("user_id", loggedInUserID), slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal mengambil data pesanan")
		return
	}

	response.Success(c, http.StatusOK, orders, pagination)
}

func (h *Handler) ListAllOrders(c *gin.Context) {
	pagination := utils.GeneratePaginationFromRequest(c)

	orders, err := h.service.ListAllOrders(c.Request.Context(), pagination)
	if err != nil {
		slog.Error("ListAllOrders gagal", slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Error server internal")
		return
	}
	response.Success(c, http.StatusOK, orders, pagination)
}

func (h *Handler) UpdateOrderStatus(c *gin.Context) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format ID Pesanan tidak valid")
		return
	}
	var req UpdateOrderStatusRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.ErrInvalidRequest, "Format status pesanan tidak valid (PENDING, PAID, SHIPPED, COMPLETED, CANCELLED)")
		return
	}
	order, err := h.service.UpdateOrderStatus(c.Request.Context(), orderID, req.Status)
	if err != nil {
		if errors.Is(err, ErrorOrderNotFound) {
			response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Pesanan tidak ditemukan")
			return
		}
		slog.Error("UpdateOrderStatus gagal", slog.Int64("order_id", orderID), slog.String("error", err.Error()))
		response.Error(c, http.StatusInternalServerError, response.ErrInternalServer, "Gagal memperbarui status pesanan")
	}
	response.Success(c, http.StatusOK, order, nil)
}
