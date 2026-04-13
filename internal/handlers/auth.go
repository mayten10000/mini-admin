package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"mini-admin/internal/middleware"
	"mini-admin/internal/models"
	"mini-admin/internal/utils"
)

type AuthHandler struct {
	DB              *sql.DB
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		utils.ErrorJSON(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	user, err := models.GetUserByEmail(h.DB, req.Email)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	if user.Status != "active" {
		utils.ErrorJSON(w, http.StatusForbidden, "Account is disabled")
		return
	}

	accessToken, err := middleware.GenerateAccessToken(h.JWTSecret, user.ID, h.AccessTokenTTL)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	refreshToken, err := models.GenerateRefreshToken()
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	if err := models.SaveRefreshToken(h.DB, user.ID, refreshToken, h.RefreshTokenTTL); err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to save refresh token")
		return
	}

	utils.JSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(h.AccessTokenTTL.Seconds()),
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		utils.ErrorJSON(w, http.StatusBadRequest, "Refresh token is required")
		return
	}

	rt, err := models.FindRefreshToken(h.DB, req.RefreshToken)
	if err != nil {
		utils.ErrorJSON(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	if time.Now().After(rt.ExpiresAt) {
		_ = models.DeleteRefreshToken(h.DB, req.RefreshToken)
		utils.ErrorJSON(w, http.StatusUnauthorized, "Refresh token expired")
		return
	}

	user, err := models.GetUserByID(h.DB, rt.UserID)
	if err != nil || user.Status != "active" {
		_ = models.DeleteRefreshToken(h.DB, req.RefreshToken)
		utils.ErrorJSON(w, http.StatusUnauthorized, "User not found or disabled")
		return
	}

	_ = models.DeleteRefreshToken(h.DB, req.RefreshToken)

	accessToken, err := middleware.GenerateAccessToken(h.JWTSecret, rt.UserID, h.AccessTokenTTL)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to generate access token")
		return
	}

	newRefresh, err := models.GenerateRefreshToken()
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	if err := models.SaveRefreshToken(h.DB, rt.UserID, newRefresh, h.RefreshTokenTTL); err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to save refresh token")
		return
	}

	utils.JSON(w, http.StatusOK, tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		ExpiresIn:    int(h.AccessTokenTTL.Seconds()),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.RefreshToken != "" {
		_ = models.DeleteRefreshToken(h.DB, req.RefreshToken)
	}

	utils.JSON(w, http.StatusOK, map[string]string{"message": "Logged out"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	userID := middleware.GetUserID(r)
	user, err := models.GetUserByID(h.DB, userID)
	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}

	utils.JSON(w, http.StatusOK, user)
}
