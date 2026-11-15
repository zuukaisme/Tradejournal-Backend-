package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeDatabase struct {
	err error
}

func (f fakeDatabase) Ping(context.Context) error {
	return f.err
}

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(fakeDatabase{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestReady(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(fakeDatabase{})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestNotReadyWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(nil)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ready", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
}
