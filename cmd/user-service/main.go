package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"time"

	googlegrpc "google.golang.org/grpc"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/cache"
	"usermanagementsystem/internal/database"
	"usermanagementsystem/internal/grpc"
	"usermanagementsystem/internal/repository"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println(" Starting User Management Microservice (gRPC)")
	fmt.Println("==================================================")

	// 1. Initialize PostgreSQL Connection Pool
	dbCfg := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgrespassword"),
		DBName:   getEnv("DB_NAME", "usermanagement"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	fmt.Printf("[DB] Connecting to PostgreSQL at %s:%s/%s...\n", dbCfg.Host, dbCfg.Port, dbCfg.DBName)
	db, err := database.NewPostgresDB(dbCfg)
	if err != nil {
		fmt.Printf("[DB] FATAL: Failed to connect to PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	fmt.Println("[DB] Connection pool established and verified successfully.")

	// 2. Initialize Redis Cache
	redisCfg := cache.Config{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6380"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	}
	fmt.Printf("[CACHE] Connecting to Redis at %s:%s...\n", redisCfg.Host, redisCfg.Port)
	userCache, err := cache.NewRedisUserCache(redisCfg)
	if err != nil {
		fmt.Printf("[CACHE] WARNING: Could not connect to Redis (%v). Proceeding without cache.\n", err)
		userCache = nil
	} else {
		fmt.Println("[CACHE] Connected to Redis successfully.")
	}

	// 3. Initialize Repositories
	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	// 4. Initialize Services (Domain Business Logic with Cache and Auth)
	jwtSecret := getEnv("JWT_SECRET", "super_secret_jwt_key_change_me_in_production")
	tokenManager := auth.NewTokenManager(jwtSecret, 24*time.Hour)

	deptService := service.NewDepartmentService(deptRepo, userRepo, userCache)
	roleService := service.NewRoleService(roleRepo, userRepo, userCache)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, userCache)
	authService := service.NewAuthService(userRepo, roleRepo, tokenManager)

	// 5. Bind TCP Listener for gRPC
	grpcPort := getEnv("GRPC_PORT", "50051")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		fmt.Printf("[GRPC] FATAL: Failed to bind port %s: %v\n", grpcPort, err)
		os.Exit(1)
	}

	// 6. Initialize and Register gRPC Server
	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.UnaryServerLoggingInterceptor()),
	)
	serverHandler := grpc.NewServer(userService, deptService, roleService, authService)

	v1.RegisterUserServiceServer(grpcServer, serverHandler)
	v1.RegisterDepartmentServiceServer(grpcServer, serverHandler)
	v1.RegisterRoleServiceServer(grpcServer, serverHandler)

	// 6. Graceful Shutdown listener
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n[GRPC] Received shutdown signal: %v. Gracefully stopping server...\n", sig)
		grpcServer.GracefulStop()
		fmt.Println("[GRPC] Server stopped cleanly.")
	}()

	fmt.Printf("[GRPC] Microservice listening on port :%s (Protocol: HTTP/2 Protobuf)\n", grpcPort)
	fmt.Println("[READY] User Management Microservice is ready to handle requests.")
	if err := grpcServer.Serve(lis); err != nil {
		fmt.Printf("[GRPC] Server terminated with error: %v\n", err)
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
