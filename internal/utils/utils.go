package utils

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

type APIError struct {
	Error   string            `json:"error"`
	Details map[string]string `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func ErrorJSON(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, APIError{Error: msg})
}

func ValidationErrorJSON(w http.ResponseWriter, details map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, APIError{
		Error:   "Validation failed",
		Details: details,
	})
}

var emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func IsValidEmail(email string) bool {
	return emailRe.MatchString(email)
}

func IsValidStatus(s string) bool {
	return s == "active" || s == "disabled"
}

func TrimAndValidateUserInput(name, email, status string) map[string]string {
	errs := map[string]string{}
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	status = strings.TrimSpace(status)

	if name == "" {
		errs["name"] = "Name is required"
	} else if len(name) > 255 {
		errs["name"] = "Name is too long (max 255)"
	}

	if email == "" {
		errs["email"] = "Email is required"
	} else if !IsValidEmail(email) {
		errs["email"] = "Invalid email format"
	}

	if status == "" {
		errs["status"] = "Status is required"
	} else if !IsValidStatus(status) {
		errs["status"] = "Status must be active or disabled"
	}

	return errs
}
