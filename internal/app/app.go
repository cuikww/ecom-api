package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecom-api/internal/auth"
	"ecom-api/internal/database"
	httpRouter "ecom-api/internal/http"
	"ecom-api/internal/orders"
	"ecom-api/internal/products"
	"ecom-api/internal/users"

	"github.com/gin-gonic/gin"
)

func Run() error {
	// ==============================
	// DATABASE
	// ==============================

	dsn := os.Getenv("GOOSE_DBSTRING")

	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=ecom port=5432 sslmode=disable"
	}

	db, err := database.Connect(dsn)
	if err != nil {
		return err
	}

	log.Println("database connected")

	redisDSN := os.Getenv("REDIS_URL")
	if redisDSN == "" {
		redisDSN = "redis://localhost:6379/0"
	}

	rdb, err := database.ConnectRedis(redisDSN)
	if err != nil {
		return err
	}
	defer rdb.Close()
	log.Println("redis connected")

	// ==============================
	// JWT
	// ==============================

	jwtSecret := os.Getenv("JWT_SECRET")
	jwtIssuer := os.Getenv("JWT_ISSUER")
	jwtAudience := os.Getenv("JWT_AUDIENCE")
	jwtExpirationString := os.Getenv("JWT_EXPIRATION")

	if jwtSecret == "" {
		return errMissingEnv("JWT_SECRET")
	}

	if jwtIssuer == "" {
		return errMissingEnv("JWT_ISSUER")
	}

	if jwtAudience == "" {
		return errMissingEnv("JWT_AUDIENCE")
	}

	jwtExpiration, err := time.ParseDuration(jwtExpirationString)
	if err != nil {
		return err
	}

	jwtService := auth.NewJWTService(
		jwtSecret,
		jwtIssuer,
		jwtAudience,
		jwtExpiration,
	)

	// ==============================
	// SERVICES
	// ==============================

	userService := users.NewService(
		db,
		jwtService,
	)

	productService := products.NewService(db, rdb)

	orderService := orders.NewService(db, rdb)

	// ==============================
	// HANDLERS
	// ==============================

	userHandler := users.NewHandler(userService)
	productHandler := products.NewHandler(productService)
	orderHandler := orders.NewHandler(orderService)

	// ==============================
	// ROUTER
	// ==============================

	router := httpRouter.NewRouter(
		userHandler,
		productHandler,
		orderHandler,
		jwtService,
	)

	// ==============================
	// SERVER (GRACEFUL SHUTDOWN)
	// ==============================

	server := gin.Default()
	router.Register(server)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: server,
	}

	go func() {
		log.Println("server running on :8080")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Menerima sinyal shutdown, mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server dipaksa mati karena timeout atau error: ", err)
	}

	log.Println("Server berhasil dimatikan secara aman (Graceful Shutdown)")
	return nil
}

type missingEnvError struct {
	key string
}

func (e missingEnvError) Error() string {
	return "missing required environment variable: " + e.key
}

func errMissingEnv(key string) error {
	return missingEnvError{key: key}
}
