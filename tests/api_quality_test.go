package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"usermanagementsystem/internal/middleware"
	"usermanagementsystem/internal/response"
)

func TestAPIQuality_RequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RequestID())
	router.GET("/test-id", func(c *gin.Context) {
		reqID := c.GetString(middleware.ContextKeyRequestID)
		response.Success(c, http.StatusOK, gin.H{"context_req_id": reqID})
	})

	// Case 1: Client provides NO X-Request-ID -> Server auto-generates a valid UUID
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodGet, "/test-id", nil)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w1.Code)
	}

	headerID := w1.Header().Get(middleware.HeaderRequestID)
	if headerID == "" {
		t.Fatal("Expected X-Request-ID response header to be set, but got empty")
	}

	if _, err := uuid.Parse(headerID); err != nil {
		t.Fatalf("Expected generated X-Request-ID to be a valid UUIDv4, got %s (err: %v)", headerID, err)
	}

	var resp1 struct {
		Success bool `json:"success"`
		Data    struct {
			ContextReqID string `json:"context_req_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if resp1.Data.ContextReqID != headerID {
		t.Fatalf("Context request ID (%s) does not match header (%s)", resp1.Data.ContextReqID, headerID)
	}

	// Case 2: Client provides a custom X-Request-ID -> Server preserves it
	customID := "client-trace-id-abc-123"
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/test-id", nil)
	req2.Header.Set(middleware.HeaderRequestID, customID)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w2.Code)
	}
	if preserved := w2.Header().Get(middleware.HeaderRequestID); preserved != customID {
		t.Fatalf("Expected preserved X-Request-ID '%s', got '%s'", customID, preserved)
	}
}

func TestAPIQuality_CORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.CORS())
	router.GET("/api/resource", func(c *gin.Context) {
		response.Success(c, http.StatusOK, gin.H{"status": "ok"})
	})

	// 1. Preflight OPTIONS request from React frontend
	wOptions := httptest.NewRecorder()
	reqOptions, _ := http.NewRequest(http.MethodOptions, "/api/resource", nil)
	reqOptions.Header.Set("Origin", "http://localhost:5173")
	reqOptions.Header.Set("Access-Control-Request-Method", "POST")
	reqOptions.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-Request-ID")
	router.ServeHTTP(wOptions, reqOptions)

	if wOptions.Code != http.StatusNoContent {
		t.Fatalf("Expected 204 No Content for OPTIONS preflight, got %d", wOptions.Code)
	}

	if allowOrigin := wOptions.Header().Get("Access-Control-Allow-Origin"); allowOrigin != "http://localhost:5173" {
		t.Fatalf("Expected Access-Control-Allow-Origin 'http://localhost:5173', got '%s'", allowOrigin)
	}

	if creds := wOptions.Header().Get("Access-Control-Allow-Credentials"); creds != "true" {
		t.Fatalf("Expected Access-Control-Allow-Credentials 'true', got '%s'", creds)
	}

	if expose := wOptions.Header().Get("Access-Control-Expose-Headers"); expose == "" {
		t.Fatal("Expected Access-Control-Expose-Headers to be set")
	}

	// 2. Standard GET request from browser origin
	wGet := httptest.NewRecorder()
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/resource", nil)
	reqGet.Header.Set("Origin", "http://localhost:5173")
	router.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", wGet.Code)
	}
	if wGet.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("Expected Allow-Origin header on GET response")
	}
}

func TestAPIQuality_PanicRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery())

	// Endpoint that intentionally panics
	router.GET("/panic-endpoint", func(c *gin.Context) {
		panic("simulated fatal database crash or nil pointer dereference")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/panic-endpoint", nil)
	router.ServeHTTP(w, req)

	// Server must NOT crash; it should return 500 Internal Server Error
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected 500 Internal Server Error from recovery, got %d", w.Code)
	}

	var errResp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("Failed to parse panic error response: %v", err)
	}

	if errResp.Success {
		t.Fatal("Expected success to be false on panic response")
	}

	if errResp.Error == nil {
		t.Fatal("Expected error object to be populated on panic response")
	}

	if errResp.Error.Code != "INTERNAL_SERVER_ERROR" {
		t.Fatalf("Expected error code INTERNAL_SERVER_ERROR, got %s", errResp.Error.Code)
	}

	if errResp.Error.RequestID == "" {
		t.Fatal("Expected RequestID to be populated in panic error envelope")
	}

	if errResp.Error.RequestID != w.Header().Get(middleware.HeaderRequestID) {
		t.Fatalf("Error envelope RequestID (%s) does not match header (%s)",
			errResp.Error.RequestID, w.Header().Get(middleware.HeaderRequestID))
	}
}

func TestAPIQuality_StandardizedErrorResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.RequestID())

	router.GET("/error-test", func(c *gin.Context) {
		response.Error(c, http.StatusBadRequest, "INVALID_PARAM", "The 'age' parameter must be positive", gin.H{"field": "age"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/error-test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request, got %d", w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.Success != false {
		t.Fatal("Expected success: false")
	}
	if resp.Error == nil {
		t.Fatal("Expected non-nil error object")
	}
	if resp.Error.Code != "INVALID_PARAM" {
		t.Fatalf("Expected code INVALID_PARAM, got %s", resp.Error.Code)
	}
	if resp.Error.RequestID == "" {
		t.Fatal("Expected RequestID in error response")
	}
}
