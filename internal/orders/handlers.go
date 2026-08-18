package orders

import (
	"errors"
	"log"
	"net/http"
	"strconv"

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
	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	loggedInUserID, ok := userIDRaw.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user token data"})
		return
	}

	var req CreateOrderRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "format request tidak valid",
		})
		return
	}
	req.CustomerID = loggedInUserID
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

		log.Printf("[ERROR] PlaceOrder gagal untuk user %d: %v", loggedInUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "gagal membuat pesanan, silakan coba lagi",
		})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) ListOrdersByCustomer(c *gin.Context) {
	userIDRaw, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	loggedInUserID, ok := userIDRaw.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user token data"})
		return
	}

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
	if customerID != loggedInUserID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "anda tidak memiliki akses ke data pesanan pengguna lain",
		})
		return
	}

	pagination := utils.GeneratePaginationFromRequest(c)

	orders, err := h.service.ListOrdersByCustomer(
		c.Request.Context(),
		customerID,
		pagination,
	)

	if err != nil {
		log.Printf("[ERROR] ListOrdersByCustomer gagal untuk user %d: %v", loggedInUserID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "gagal mengambil data pesanan, silakan coba lagi",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"meta": pagination,
		"data": orders,
	})
}
