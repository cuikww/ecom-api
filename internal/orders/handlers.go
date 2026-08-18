package orders

import (
	"errors"
	"net/http"
	"strconv"

	"ecom-api/internal/utils" // <-- Sesuaikan dengan path project Anda

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
	var req CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	order, err := h.service.PlaceOrder(
		c.Request.Context(),
		req,
	)

	if err != nil {
		if errors.Is(err, ErrorProductNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "product not found",
			})
			return
		}

		if errors.Is(err, ErrorProductNoStock) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "product has insufficient stock",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create order",
		})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) ListOrdersByCustomer(c *gin.Context) {
	customerID, err := strconv.ParseInt(
		c.Param("customer_id"),
		10,
		64,
	)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid customer id",
		})
		return
	}

	// 1. Ambil meta paginasi dari query params
	pagination := utils.GeneratePaginationFromRequest(c)

	// 2. Teruskan pagination ke service
	orders, err := h.service.ListOrdersByCustomer(
		c.Request.Context(),
		customerID,
		pagination,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meta": pagination,
		"data": orders,
	})
}
