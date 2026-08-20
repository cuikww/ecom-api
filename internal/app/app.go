package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecom-api/internal/auth"
	"ecom-api/internal/database"
	httpRouter "ecom-api/internal/http"
	"ecom-api/internal/logger"
	"ecom-api/internal/orders"
	"ecom-api/internal/products"
	"ecom-api/internal/users"
	"ecom-api/internal/worker"

	"github.com/gin-gonic/gin"
)

func Run() error {
	logger.InitLogger()
	slog.Info("Memulai aplikasi e-commerce API...")

	// ==============================
	// DATABASE
	// ==============================
	dsn := os.Getenv("GOOSE_DBSTRING")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=ecom port=5432 sslmode=disable"
	}

	db, err := database.Connect(dsn)
	if err != nil {
		slog.Error("Gagal koneksi database", slog.String("error", err.Error()))
		return err
	}
	slog.Info("Database PostgreSQL terhubung")

	err = db.AutoMigrate(
		&users.User{},
		&products.Product{},
		&orders.Order{},
		&orders.OrderItem{},
	)
	if err != nil {
		slog.Error("Gagal migrate database", slog.String("error", err.Error()))
		return err
	}
	slog.Info("Database schema synced successfully") // Diganti menggunakan slog

	redisDSN := os.Getenv("REDIS_URL")
	if redisDSN == "" {
		redisDSN = "redis://localhost:6379/0"
	}

	rdb, err := database.ConnectRedis(redisDSN)
	if err != nil {
		slog.Error("Gagal koneksi Redis", slog.String("error", err.Error()))
		return err
	}
	defer rdb.Close()
	slog.Info("Redis terhubung")

	// ==============================
	// Worker
	// ==============================

	workerPool := worker.NewPool(5, 1000)

	bgCtx, cancelBg := context.WithCancel(context.Background())
	defer cancelBg()

	workerPool.Start(bgCtx)

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

	productService := products.NewService(db, rdb, workerPool)
	orderService := orders.NewService(db, rdb, workerPool)

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

	gin.SetMode(gin.ReleaseMode)
	server := gin.New()
	router.Register(server)

	srv := &http.Server{
		Addr:    ":8080",
		Handler: server,
	}

	go func() {
		slog.Info("Server berjalan", slog.String("port", "8080"))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server gagal berjalan", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Menerima sinyal shutdown, mematikan server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server dipaksa mati", slog.String("error", err.Error()))
	}

	workerPool.Stop()

	slog.Info("Server berhasil dimatikan secara aman")
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
