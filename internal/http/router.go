package http

import (
	"ecom-api/internal/auth"
	"ecom-api/internal/orders"
	"ecom-api/internal/products"
	"ecom-api/internal/users"

	"github.com/gin-gonic/gin"
)

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
		c.JSON(200, gin.H{
			"message": "All Good! Yeaaay",
		})
	})

	// ==================================
	// PROTECTED
	// ==================================

	protected := r.Group("/")
	protected.Use(
		auth.AuthMiddleware(rt.jwtService),
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

	protected.POST(
		"/products",
		rt.productHandler.CreateProduct,
	)

	protected.PATCH(
		"/products/:id",
		rt.productHandler.UpdateProduct,
	)

	// Orders
	protected.POST(
		"/orders",
		rt.orderHandler.PlaceOrder,
	)

	protected.GET(
		"/orders/customer/:customer_id",
		rt.orderHandler.ListOrdersByCustomer,
	)
}
