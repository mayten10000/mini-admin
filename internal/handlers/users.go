package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"mini-admin/internal/models"
	"mini-admin/internal/utils"
)

type UserHandler struct {
	DB *sql.DB
}

type createUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

type updateUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
	Password string `json:"password,omitempty"`
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path == "/api/users" {
		switch r.Method {
		case http.MethodGet:
			h.List(w, r)
		case http.MethodPost:
			h.Create(w, r)
		default:
			utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	prefix := "/api/users/"
	if strings.HasPrefix(r.URL.Path, prefix) {
		idStr := strings.TrimPrefix(r.URL.Path, prefix)
		idStr = strings.TrimSuffix(idStr, "/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			utils.ErrorJSON(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.Get(w, r, id)
		case http.MethodPut:
			h.Update(w, r, id)
		case http.MethodDelete:
			h.Delete(w, r, id)
		default:
			utils.ErrorJSON(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	utils.ErrorJSON(w, http.StatusNotFound, "Not found")
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	params := models.UserListParams{
		Search:  q.Get("search"),
		Status:  q.Get("status"),
		SortBy:  q.Get("sort_by"),
		Order:   q.Get("order"),
		Page:    page,
		PerPage: perPage,
	}

	result, err := models.ListUsers(h.DB, params)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to list users")
		return
	}

	utils.JSON(w, http.StatusOK, result)
}

func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request, id int64) {
	user, err := models.GetUserByID(h.DB, id)
	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}
	utils.JSON(w, http.StatusOK, user)
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Status = strings.TrimSpace(req.Status)

	if req.Status == "" {
		req.Status = "active"
	}

	errs := utils.TrimAndValidateUserInput(req.Name, req.Email, req.Status)
	if req.Password == "" {
		errs["password"] = "Password is required"
	} else if len(req.Password) < 6 {
		errs["password"] = "Password must be at least 6 characters"
	}

	if len(errs) > 0 {
		utils.ValidationErrorJSON(w, errs)
		return
	}

	if _, err := models.GetUserByEmail(h.DB, req.Email); err == nil {
		utils.ValidationErrorJSON(w, map[string]string{"email": "Email already taken"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hash),
		Status:   req.Status,
	}

	if err := models.CreateUser(h.DB, user); err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	utils.JSON(w, http.StatusCreated, user)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request, id int64) {
	existing, err := models.GetUserByID(h.DB, id)
	if err != nil {
		utils.ErrorJSON(w, http.StatusNotFound, "User not found")
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorJSON(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	req.Status = strings.TrimSpace(req.Status)

	errs := utils.TrimAndValidateUserInput(req.Name, req.Email, req.Status)
	if req.Password != "" && len(req.Password) < 6 {
		errs["password"] = "Password must be at least 6 characters"
	}

	if len(errs) > 0 {
		utils.ValidationErrorJSON(w, errs)
		return
	}

	if req.Email != existing.Email {
		if other, err := models.GetUserByEmail(h.DB, req.Email); err == nil && other.ID != id {
			utils.ValidationErrorJSON(w, map[string]string{"email": "Email already taken"})
			return
		}
	}

	existing.Name = req.Name
	existing.Email = req.Email
	existing.Status = req.Status

	if err := models.UpdateUser(h.DB, existing); err != nil {
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}
		if err := models.UpdateUserPassword(h.DB, id, string(hash)); err != nil {
			utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to update password")
			return
		}
	}

	utils.JSON(w, http.StatusOK, existing)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request, id int64) {
	if err := models.DeleteUser(h.DB, id); err != nil {
		if err == sql.ErrNoRows {
			utils.ErrorJSON(w, http.StatusNotFound, "User not found")
			return
		}
		utils.ErrorJSON(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"message": "User deleted"})
}
