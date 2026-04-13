package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserListParams struct {
	Search  string
	Status  string
	SortBy  string
	Order   string
	Page    int
	PerPage int
}

type UserListResult struct {
	Users      []User `json:"users"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PerPage    int    `json:"per_page"`
	TotalPages int    `json:"total_pages"`
}

func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	u := &User{}
	err := db.QueryRow(
		"SELECT id, name, email, password, status, created_at, updated_at FROM users WHERE email=$1",
		email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func GetUserByID(db *sql.DB, id int64) (*User, error) {
	u := &User{}
	err := db.QueryRow(
		"SELECT id, name, email, password, status, created_at, updated_at FROM users WHERE id=$1",
		id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func ListUsers(db *sql.DB, p UserListParams) (*UserListResult, error) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 || p.PerPage > 100 {
		p.PerPage = 20
	}

	allowedSort := map[string]bool{"id": true, "name": true, "email": true, "status": true, "created_at": true, "updated_at": true}
	if !allowedSort[p.SortBy] {
		p.SortBy = "id"
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "asc"
	}

	where := []string{"1=1"}
	args := []interface{}{}
	idx := 1

	if p.Search != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR email ILIKE $%d)", idx, idx))
		args = append(args, "%"+p.Search+"%")
		idx++
	}
	if p.Status != "" {
		where = append(where, fmt.Sprintf("status=$%d", idx))
		args = append(args, p.Status)
		idx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQ := "SELECT COUNT(*) FROM users WHERE " + whereClause
	if err := db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	totalPages := (total + p.PerPage - 1) / p.PerPage
	offset := (p.Page - 1) * p.PerPage

	query := fmt.Sprintf(
		"SELECT id, name, email, password, status, created_at, updated_at FROM users WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		whereClause, p.SortBy, p.Order, idx, idx+1,
	)
	args = append(args, p.PerPage, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Password, &u.Status, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.Password = ""
		users = append(users, u)
	}

	return &UserListResult{
		Users:      users,
		Total:      total,
		Page:       p.Page,
		PerPage:    p.PerPage,
		TotalPages: totalPages,
	}, nil
}

func CreateUser(db *sql.DB, u *User) error {
	return db.QueryRow(
		`INSERT INTO users (name, email, password, status)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at, updated_at`,
		u.Name, u.Email, u.Password, u.Status,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func UpdateUser(db *sql.DB, u *User) error {
	return db.QueryRow(
		`UPDATE users SET name=$1, email=$2, status=$3, updated_at=now()
		 WHERE id=$4
		 RETURNING updated_at`,
		u.Name, u.Email, u.Status, u.ID,
	).Scan(&u.UpdatedAt)
}

func UpdateUserPassword(db *sql.DB, id int64, hash string) error {
	_, err := db.Exec("UPDATE users SET password=$1, updated_at=now() WHERE id=$2", hash, id)
	return err
}

func DeleteUser(db *sql.DB, id int64) error {
	res, err := db.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
