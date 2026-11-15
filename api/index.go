package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zuukaisme/Tradejournal-Backend-/internal/config"
	"github.com/zuukaisme/Tradejournal-Backend-/internal/database"
	"github.com/zuukaisme/Tradejournal-Backend-/internal/httpapi"
)

var (
	initMu sync.Mutex
	appHandler http.Handler
)

func Handler(w http.ResponseWriter, r *http.Request) {
	initMu.Lock()
	if appHandler == nil {
		cfg, err := config.Load()
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			pool, connectErr := database.Connect(ctx, cfg.DatabaseURL)
			if connectErr != nil {
				err = connectErr
			} else if migrateErr := database.Migrate(ctx, pool); migrateErr != nil {
				pool.Close()
				err = migrateErr
			} else {
				if cfg.Environment == "production" {
					gin.SetMode(gin.ReleaseMode)
				}
				appHandler = httpapi.NewRouter(pool, cfg.JWTSecret)
			}
			cancel()
		}
		if err != nil {
			initMu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Service temporarily unavailable"})
			return
		}
	}
	handler := appHandler
	initMu.Unlock()

	path := r.URL.Query().Get("path")
	if path != "" {
		r.URL.Path = "/" + strings.TrimLeft(path, "/")
		r.URL.RawPath = ""
		query := r.URL.Query()
		query.Del("path")
		r.URL.RawQuery = query.Encode()
	}
	handler.ServeHTTP(w, r)
}
