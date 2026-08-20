package http

import (
	"ecom-api/internal/auth"
	"ecom-api/internal/orders"
	"ecom-api/internal/products"
	"ecom-api/internal/response"
	"ecom-api/internal/users"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next() // Proses request ke handler

		latency := time.Since(start)
		status := c.Writer.Status()

		slog.Info("HTTP Request",
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.String("latency", latency.String()),
			slog.String("ip", c.ClientIP()),
		)
	}
}

type Router struct {
	userHandler    *users.Handler
	productHandler *products.Handler
	orderHandler   *orders.Handler
	jwtService     *auth.JWTService
}

func NewRouter(
	userHandler *users.Handler,
	productHandler *products.Handler,
	orderHandler *orders.Handler,
	jwtService *auth.JWTService,
) *Router {
	return &Router{
		userHandler:    userHandler,
		productHandler: productHandler,
		orderHandler:   orderHandler,
		jwtService:     jwtService,
	}
}

func (rt *Router) Register(r *gin.Engine) {
	r.Use(RequestLogger(), gin.Recovery())

	// ==================================
	// PUBLIC
	// ==================================

	r.POST(
		"/users",
		rt.userHandler.CreateUser,
	)

	r.POST(
		"/users/login",
		rt.userHandler.LoginUser,
	)

	r.GET("/health", func(c *gin.Context) {
		response.Success(c, 200, gin.H{"status": "API is healthy"}, nil)
	})

	// ==================================
	// PROTECTED
	// ==================================

	protected := r.Group("/")
	protected.Use(
		auth.AuthMiddleware(rt.jwtService),
	)
	//User
	protected.GET(
		"/users",
		rt.userHandler.GetProfile,
	)
	protected.PATCH(
		"/users",
		rt.userHandler.UpdateProfile,
	)

	// Products
	protected.GET(
		"/products",
		rt.productHandler.ListProducts,
	)

	protected.GET(
		"/products/:id",
		rt.productHandler.FindProductsByID,
	)

	adminRoutes := protected.Group("/")
	adminRoutes.Use(auth.RoleMiddleware(users.RoleAdmin))

	adminRoutes.POST(
		"/products",
		rt.productHandler.CreateProduct,
	)

	adminRoutes.PATCH(
		"/products/:id",
		rt.productHandler.UpdateProduct,
	)

	// Orders
	adminRoutes.GET(
		"/orders",
		rt.orderHandler.ListAllOrders,
	)
	adminRoutes.PATCH(
		"/orders/:id/status",
		rt.orderHandler.UpdateOrderStatus,
	)

	protected.POST(
		"/orders",
		rt.orderHandler.PlaceOrder,
	)

	protected.GET(
		"/orders/customer/:customer_id",
		rt.orderHandler.ListOrdersByCustomer,
	)
}
