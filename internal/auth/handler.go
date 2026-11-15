package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	db *pgxpool.Pool
	secret []byte
}

type credentials struct {
	Email string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	DeviceName string `json:"device_name" binding:"omitempty,max=120"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type claims struct {
	UserID string `json:"sub"`
	AccountID string `json:"account_id"`
	ExpiresAt int64 `json:"exp"`
	IssuedAt int64 `json:"iat"`
	Issuer string `json:"iss"`
}

func NewHandler(db *pgxpool.Pool, secret string) *Handler {
	return &Handler{db: db, secret: []byte(secret)}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/register", h.register)
	router.POST("/login", h.login)
	router.POST("/refresh", h.refresh)
	router.POST("/logout", h.requireAuth(), h.logout)
	router.GET("/me", h.requireAuth(), h.me)
}

func (h *Handler) register(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a valid email and a password of at least 8 characters"})
		return
	}
	email := strings.TrimSpace(input.Email)
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}
	var userID, accountID, savedEmail string
	err = h.db.QueryRow(c.Request.Context(), `INSERT INTO users (email, email_normalized, password_hash) VALUES ($1, lower($1), $2) RETURNING id, account_id, email`, email, string(hash)).Scan(&userID, &accountID, &savedEmail)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "An account with this email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create account"})
		return
	}
	h.createSession(c, userID, accountID, savedEmail, input.DeviceName, http.StatusCreated)
}

func (h *Handler) login(c *gin.Context) {
	var input credentials
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a valid email and password"})
		return
	}
	var userID, accountID, email, passwordHash string
	err := h.db.QueryRow(c.Request.Context(), `SELECT id, account_id, email, password_hash FROM users WHERE email_normalized = lower($1)`, strings.TrimSpace(input.Email)).Scan(&userID, &accountID, &email, &passwordHash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	h.createSession(c, userID, accountID, email, input.DeviceName, http.StatusOK)
}

func (h *Handler) refresh(c *gin.Context) {
	var input refreshRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Refresh token is required"})
		return
	}
	oldHash := tokenHash(input.RefreshToken)
	tx, err := h.db.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not refresh session"})
		return
	}
	defer tx.Rollback(c.Request.Context())
	var sessionID, userID, accountID, email string
	err = tx.QueryRow(c.Request.Context(), `SELECT s.id, u.id, u.account_id, u.email FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.refresh_token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > NOW() FOR UPDATE OF s`, oldHash).Scan(&sessionID, &userID, &accountID, &email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token is invalid or expired"})
		return
	}
	raw, err := randomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not refresh session"})
		return
	}
	newHash := tokenHash(raw)
	_, err = tx.Exec(c.Request.Context(), `UPDATE sessions SET refresh_token_hash = $1, expires_at = NOW() + INTERVAL '30 days', last_used_at = NOW() WHERE id = $2`, newHash, sessionID)
	if err != nil || tx.Commit(c.Request.Context()) != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not refresh session"})
		return
	}
	access, expiry, err := h.accessToken(userID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not refresh session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"access_token": access, "refresh_token": raw, "token_type": "Bearer", "expires_in": expiry})
}

func (h *Handler) createSession(c *gin.Context, userID, accountID, email, device string, status int) {
	raw, err := randomToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create session"})
		return
	}
	_, err = h.db.Exec(c.Request.Context(), `INSERT INTO sessions (user_id, refresh_token_hash, device_name, expires_at, last_used_at) VALUES ($1, $2, $3, NOW() + INTERVAL '30 days', NOW())`, userID, tokenHash(raw), nullableString(device))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create session"})
		return
	}
	access, expiry, err := h.accessToken(userID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create session"})
		return
	}
	c.JSON(status, gin.H{"user": gin.H{"id": userID, "account_id": accountID, "email": email}, "access_token": access, "refresh_token": raw, "token_type": "Bearer", "expires_in": expiry})
}

func (h *Handler) logout(c *gin.Context) {
	userID := c.GetString("user_id")
	_, err := h.db.Exec(c.Request.Context(), `UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not log out"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("user_id"), "account_id": c.GetString("account_id")})
}

func (h *Handler) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		claims, err := h.parseAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Access token is invalid or expired"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("account_id", claims.AccountID)
		c.Next()
	}
}

func (h *Handler) accessToken(userID, accountID string) (string, int64, error) {
	now := time.Now()
	exp := now.Add(15 * time.Minute).Unix()
	payload, err := json.Marshal(claims{UserID: userID, AccountID: accountID, ExpiresAt: exp, IssuedAt: now.Unix(), Issuer: "tradejournal-api"})
	if err != nil {
		return "", 0, err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	unsigned := header + "." + body
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + signature, 900, nil
}

func (h *Handler) parseAccessToken(token string) (claims, error) {
	var result claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return result, errors.New("invalid token")
	}
	expectedHeader := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	if parts[0] != expectedHeader {
		return result, errors.New("unsupported token header")
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := mac.Sum(nil)
	actual, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(expected, actual) {
		return result, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || json.Unmarshal(payload, &result) != nil {
		return result, errors.New("invalid claims")
	}
	now := time.Now().Unix()
	if result.ExpiresAt <= now || result.IssuedAt > now+60 || result.ExpiresAt <= result.IssuedAt || result.ExpiresAt-result.IssuedAt > 15*60 || result.Issuer != "tradejournal-api" || result.UserID == "" || result.AccountID == "" {
		return result, errors.New("expired or invalid claims")
	}
	return result, nil
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func tokenHash(value string) string {
	hash := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

