package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CustomDomain string    `json:"custom_domain"`
	StorageUsed  int64     `json:"storage_used"`
	MaxStorage   int64     `json:"max_storage"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func CreateUser(db *Database, username, email, passwordHash string) (int64, error) {
	query := `
	INSERT INTO users (username, email, password_hash, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?)
	`
	
	result, err := db.Exec(query, username, email, passwordHash, time.Now(), time.Now())
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

func GetUserByID(db *Database, id int) (*User, error) {
	var user User
	query := `
	SELECT id, username, email, password_hash, is_admin, custom_domain,
	       storage_used, max_storage, is_active, created_at, updated_at
	FROM users WHERE id = ?
	`
	
	err := db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsAdmin, &user.CustomDomain, &user.StorageUsed, &user.MaxStorage,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func GetUserByEmail(db *Database, email string) (*User, error) {
	var user User
	query := `
	SELECT id, username, email, password_hash, is_admin, custom_domain,
	       storage_used, max_storage, is_active, created_at, updated_at
	FROM users WHERE email = ?
	`
	
	err := db.QueryRow(query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsAdmin, &user.CustomDomain, &user.StorageUsed, &user.MaxStorage,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func GetUserByUsername(db *Database, username string) (*User, error) {
	var user User
	query := `
	SELECT id, username, email, password_hash, is_admin, custom_domain,
	       storage_used, max_storage, is_active, created_at, updated_at
	FROM users WHERE username = ?
	`
	
	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.IsAdmin, &user.CustomDomain, &user.StorageUsed, &user.MaxStorage,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

func UpdateUser(db *Database, user *User) error {
	query := `
	UPDATE users SET
		username = ?,
		email = ?,
		is_admin = ?,
		custom_domain = ?,
		storage_used = ?,
		max_storage = ?,
		is_active = ?,
		updated_at = ?
	WHERE id = ?
	`
	
	_, err := db.Exec(query,
		user.Username, user.Email, user.IsAdmin, user.CustomDomain,
		user.StorageUsed, user.MaxStorage, user.IsActive, time.Now(), user.ID,
	)
	
	return err
}

func DeleteUser(db *Database, id int) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func GetAllUsers(db *Database, limit, offset int) ([]*User, error) {
	query := `
	SELECT id, username, email, is_admin, custom_domain,
	       storage_used, max_storage, is_active, created_at, updated_at
	FROM users
	ORDER BY id DESC
	LIMIT ? OFFSET ?
	`
	
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email,
			&user.IsAdmin, &user.CustomDomain, &user.StorageUsed,
			&user.MaxStorage, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	
	return users, nil
}

func CountUsers(db *Database) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

func UpdateUserStorage(db *Database, userID int, storageUsed int64) error {
	query := `UPDATE users SET storage_used = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, storageUsed, time.Now(), userID)
	return err
}

func UpdateCustomDomain(db *Database, userID int, domain string) error {
	query := `UPDATE users SET custom_domain = ?, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, domain, time.Now(), userID)
	return err
}