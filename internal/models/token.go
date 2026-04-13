package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

type RefreshToken struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func SaveRefreshToken(db *sql.DB, userID int64, token string, ttl time.Duration) error {
	_, err := db.Exec(
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES ($1, $2, $3)",
		userID, token, time.Now().Add(ttl),
	)
	return err
}

func FindRefreshToken(db *sql.DB, token string) (*RefreshToken, error) {
	rt := &RefreshToken{}
	err := db.QueryRow(
		"SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token=$1",
		token,
	).Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt)
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func DeleteRefreshToken(db *sql.DB, token string) error {
	_, err := db.Exec("DELETE FROM refresh_tokens WHERE token=$1", token)
	return err
}

func DeleteUserRefreshTokens(db *sql.DB, userID int64) error {
	_, err := db.Exec("DELETE FROM refresh_tokens WHERE user_id=$1", userID)
	return err
}
