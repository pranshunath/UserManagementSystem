package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("🚀 User Management System - Multi-Service Launcher")
	fmt.Println("==================================================")
	fmt.Println("1. Launching User Management Microservice (gRPC :50051)...")

	// Start user-service in a sub-process
	grpcCmd := exec.Command("go", "run", "cmd/user-service/main.go")
	grpcCmd.Stdout = os.Stdout
	grpcCmd.Stderr = os.Stderr
	if err := grpcCmd.Start(); err != nil {
		fmt.Printf("Failed to start gRPC microservice: %v\n", err)
		os.Exit(1)
	}

	// Wait 2 seconds for gRPC server to bind and connect to PostgreSQL
	time.Sleep(2 * time.Second)

	fmt.Println("2. Launching API Gateway (HTTP/REST :8080)...")
	apiCmd := exec.Command("go", "run", "cmd/api/main.go")
	apiCmd.Stdout = os.Stdout
	apiCmd.Stderr = os.Stderr
	if err := apiCmd.Start(); err != nil {
		fmt.Printf("Failed to start REST API Gateway: %v\n", err)
		_ = grpcCmd.Process.Kill()
		os.Exit(1)
	}

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nStopping all services...")
	_ = apiCmd.Process.Kill()
	_ = grpcCmd.Process.Kill()
	fmt.Println("All services stopped.")
}
