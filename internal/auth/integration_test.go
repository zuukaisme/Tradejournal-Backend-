package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zuukaisme/Tradejournal-Backend-/internal/database"
)

func TestAuthLifecycleWithPostgres(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, "integration-test-secret-that-is-long-enough").RegisterRoutes(router.Group("/v1/auth"))

	email := "integration-" + tokenHash(time.Now().String())[:16] + "@example.com"
	registerBody, _ := json.Marshal(map[string]string{
		"email": email,
		"password": "correct-horse-battery",
		"device_name": "CI test device",
	})
	register := httptest.NewRecorder()
	router.ServeHTTP(register, httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewReader(registerBody)))
	if register.Code != http.StatusCreated {
		t.Fatalf("registration returned %d: %s", register.Code, register.Body.String())
	}

	var registration struct {
		User struct {
			AccountID string `json:"account_id"`
		} `json:"user"`
		AccessToken string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(register.Body.Bytes(), &registration); err != nil {
		t.Fatalf("decode registration: %v", err)
	}
	if registration.User.AccountID == "" || registration.AccessToken == "" || registration.RefreshToken == "" {
		t.Fatal("registration response is missing account or token data")
	}

	me := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+registration.AccessToken)
	router.ServeHTTP(me, meRequest)
	if me.Code != http.StatusOK {
		t.Fatalf("authenticated account lookup returned %d: %s", me.Code, me.Body.String())
	}

	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": registration.RefreshToken})
	refresh := httptest.NewRecorder()
	router.ServeHTTP(refresh, httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(refreshBody)))
	if refresh.Code != http.StatusOK {
		t.Fatalf("refresh returned %d: %s", refresh.Code, refresh.Body.String())
	}
	var rotated struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(refresh.Body.Bytes(), &rotated); err != nil || rotated.RefreshToken == "" {
		t.Fatal("refresh did not return a rotated token")
	}

	replay := httptest.NewRecorder()
	router.ServeHTTP(replay, httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(refreshBody)))
	if replay.Code != http.StatusUnauthorized {
		t.Fatalf("replayed refresh token returned %d, expected 401", replay.Code)
	}

	logout := httptest.NewRecorder()
	logoutRequest := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	logoutRequest.Header.Set("Authorization", "Bearer "+registration.AccessToken)
	router.ServeHTTP(logout, logoutRequest)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout returned %d: %s", logout.Code, logout.Body.String())
	}

	rotatedBody, _ := json.Marshal(map[string]string{"refresh_token": rotated.RefreshToken})
	afterLogout := httptest.NewRecorder()
	router.ServeHTTP(afterLogout, httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", bytes.NewReader(rotatedBody)))
	if afterLogout.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after logout returned %d, expected 401", afterLogout.Code)
	}
}
