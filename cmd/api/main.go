package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/grpc"
	"usermanagementsystem/internal/handler"
	"usermanagementsystem/internal/middleware"
	"usermanagementsystem/internal/response"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("🌐 Starting API Gateway / REST Server (Gin)")
	fmt.Println("==================================================")

	grpcTarget := getEnv("GRPC_TARGET", "localhost:50051")
	fmt.Printf("[GATEWAY] Connecting to internal gRPC Microservice at %s...\n", grpcTarget)

	grpcClient, err := grpc.NewClient(grpcTarget)
	if err != nil {
		fmt.Printf("[GATEWAY] FATAL: Failed to connect to gRPC server: %v\n", err)
		os.Exit(1)
	}
	defer grpcClient.Close()

	fmt.Println("[GATEWAY] gRPC client connected and ready.")

	// Initialize Gin engine
	router := gin.New()

	// Production-grade middlewares
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.Recovery())

	// Global health check
	router.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, gin.H{
			"status":  "ok",
			"service": "api-gateway",
			"time":    time.Now().UTC(),
		})
	})

	// Handlers with injected gRPC client stubs.
	// REST API never directly accesses the database.
	userHandler := handler.NewUserHandler(grpcClient.User)
	deptHandler := handler.NewDepartmentHandler(grpcClient.Department)
	roleHandler := handler.NewRoleHandler(grpcClient.Role)
	authHandler := handler.NewAuthHandler(grpcClient.User)

	jwtSecret := getEnv(
		"JWT_SECRET",
		"super_secret_jwt_key_change_me_in_production",
	)

	tokenManager := auth.NewTokenManager(jwtSecret, 24*time.Hour)

	// API v1 Routes
	v1 := router.Group("/api/v1")
	{
		// ==================================================
		// Public Authentication Routes
		// ==================================================

		// Login
		v1.POST("/auth/login", authHandler.Login)

		// Registration
		v1.POST("/auth/register", authHandler.Register)

		// ==================================================
		// Protected Routes
		// ==================================================

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(tokenManager))

		{
			// Users:
			// Read = Any authenticated user
			// Create/Update = Admin or Manager
			// Delete = Admin only
			users := protected.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.GET("/:id", userHandler.GetUser)

				users.POST(
					"",
					middleware.RequireRole("Admin", "Manager"),
					userHandler.CreateUser,
				)

				users.PUT(
					"/:id",
					middleware.RequireRole("Admin", "Manager"),
					userHandler.UpdateUser,
				)

				users.DELETE(
					"/:id",
					middleware.RequireRole("Admin"),
					userHandler.DeleteUser,
				)
			}

			// Departments:
			// Read = Authenticated
			// Create/Update/Delete = Admin only
			departments := protected.Group("/departments")
			{
				departments.GET("", deptHandler.ListDepartments)
				departments.GET("/:id", deptHandler.GetDepartment)

				departments.POST(
					"",
					middleware.RequireRole("Admin"),
					deptHandler.CreateDepartment,
				)

				departments.PUT(
					"/:id",
					middleware.RequireRole("Admin"),
					deptHandler.UpdateDepartment,
				)

				departments.DELETE(
					"/:id",
					middleware.RequireRole("Admin"),
					deptHandler.DeleteDepartment,
				)
			}

			// Roles:
			// Read = Authenticated
			// Create/Update/Delete = Admin only
			roles := protected.Group("/roles")
			{
				roles.GET("", roleHandler.ListRoles)
				roles.GET("/:id", roleHandler.GetRole)

				roles.POST(
					"",
					middleware.RequireRole("Admin"),
					roleHandler.CreateRole,
				)

				roles.PUT(
					"/:id",
					middleware.RequireRole("Admin"),
					roleHandler.UpdateRole,
				)

				roles.DELETE(
					"/:id",
					middleware.RequireRole("Admin"),
					roleHandler.DeleteRole,
				)
			}
		}
	}

	httpPort := getEnv("SERVER_PORT", "8080")

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	// Graceful Shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		<-sigChan

		fmt.Println(
			"\n[HTTP] Received shutdown signal. Gracefully shutting down REST API...",
		)

		ctx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		_ = srv.Shutdown(ctx)

		fmt.Println("[HTTP] REST API stopped cleanly.")
	}()

	fmt.Printf(
		"[HTTP] API Gateway listening on http://localhost:%s\n",
		httpPort,
	)

	fmt.Println("[READY] REST API Gateway is ready to serve HTTP traffic.")

	if err := srv.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		fmt.Printf(
			"[HTTP] Server terminated with error: %v\n",
			err,
		)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}

	return defaultVal
}
