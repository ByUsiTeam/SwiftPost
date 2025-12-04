package models

import (
	"database/sql"
	"time"
	"github.com/google/uuid"
)

type Email struct {
	ID              int       `json:"id"`
	UUID            string    `json:"uuid"`
	SenderID        int       `json:"sender_id"`
	RecipientID     int       `json:"recipient_id"`
	SenderEmail     string    `json:"sender_email"`
	RecipientEmail  string    `json:"recipient_email"`
	SenderName      string    `json:"sender_name"`
	RecipientName   string    `json:"recipient_name"`
	Subject         string    `json:"subject"`
	Body            string    `json:"body"`
	IsRead          bool      `json:"is_read"`
	IsStarred       bool      `json:"is_starred"`
	IsDeleted       bool      `json:"is_deleted"`
	IsDraft         bool      `json:"is_draft"`
	HasAttachment   bool      `json:"has_attachment"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type EmailWithDetails struct {
	Email
	Attachments []Attachment `json:"attachments,omitempty"`
}

func CreateEmail(db *Database, email *Email) (int64, error) {
	if email.UUID == "" {
		email.UUID = uuid.New().String()
	}
	
	query := `
	INSERT INTO emails (
		uuid, sender_id, recipient_id, sender_email, recipient_email,
		subject, body, is_read, is_starred, is_deleted, is_draft,
		has_attachment, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := db.Exec(query,
		email.UUID, email.SenderID, email.RecipientID,
		email.SenderEmail, email.RecipientEmail,
		email.Subject, email.Body,
		email.IsRead, email.IsStarred, email.IsDeleted, email.IsDraft,
		email.HasAttachment, time.Now(), time.Now(),
	)
	
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

func GetEmailByID(db *Database, id int) (*Email, error) {
	var email Email
	query := `
	SELECT id, uuid, sender_id, recipient_id, sender_email, recipient_email,
	       subject, body, is_read, is_starred, is_deleted, is_draft,
	       has_attachment, created_at, updated_at
	FROM emails WHERE id = ?
	`
	
	err := db.QueryRow(query, id).Scan(
		&email.ID, &email.UUID, &email.SenderID, &email.RecipientID,
		&email.SenderEmail, &email.RecipientEmail,
		&email.Subject, &email.Body,
		&email.IsRead, &email.IsStarred, &email.IsDeleted, &email.IsDraft,
		&email.HasAttachment, &email.CreatedAt, &email.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &email, nil
}

func GetEmailByUUID(db *Database, emailUUID string) (*Email, error) {
	var email Email
	query := `
	SELECT id, uuid, sender_id, recipient_id, sender_email, recipient_email,
	       subject, body, is_read, is_starred, is_deleted, is_draft,
	       has_attachment, created_at, updated_at
	FROM emails WHERE uuid = ?
	`
	
	err := db.QueryRow(query, emailUUID).Scan(
		&email.ID, &email.UUID, &email.SenderID, &email.RecipientID,
		&email.SenderEmail, &email.RecipientEmail,
		&email.Subject, &email.Body,
		&email.IsRead, &email.IsStarred, &email.IsDeleted, &email.IsDraft,
		&email.HasAttachment, &email.CreatedAt, &email.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &email, nil
}

func GetEmailsByRecipient(db *Database, recipientID int, limit, offset int, folder string) ([]*Email, error) {
	var query string
	var args []interface{}
	
	switch folder {
	case "inbox":
		query = `
		SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
		       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
		       e.has_attachment, e.created_at, e.updated_at
		FROM emails e
		WHERE e.recipient_id = ? AND e.is_deleted = 0 AND e.is_draft = 0
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
		`
		args = []interface{}{recipientID, limit, offset}
	case "sent":
		query = `
		SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
		       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
		       e.has_attachment, e.created_at, e.updated_at
		FROM emails e
		WHERE e.sender_id = ? AND e.is_deleted = 0 AND e.is_draft = 0
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
		`
		args = []interface{}{recipientID, limit, offset}
	case "starred":
		query = `
		SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
		       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
		       e.has_attachment, e.created_at, e.updated_at
		FROM emails e
		WHERE (e.sender_id = ? OR e.recipient_id = ?) 
		  AND e.is_starred = 1 AND e.is_deleted = 0
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
		`
		args = []interface{}{recipientID, recipientID, limit, offset}
	case "drafts":
		query = `
		SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
		       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
		       e.has_attachment, e.created_at, e.updated_at
		FROM emails e
		WHERE e.sender_id = ? AND e.is_draft = 1 AND e.is_deleted = 0
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
		`
		args = []interface{}{recipientID, limit, offset}
	case "trash":
		query = `
		SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
		       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
		       e.has_attachment, e.created_at, e.updated_at
		FROM emails e
		WHERE (e.sender_id = ? OR e.recipient_id = ?) AND e.is_deleted = 1
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
		`
		args = []interface{}{recipientID, recipientID, limit, offset}
	default: // inbox
		query = `
		SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
		       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
		       e.has_attachment, e.created_at, e.updated_at
		FROM emails e
		WHERE e.recipient_id = ? AND e.is_deleted = 0 AND e.is_draft = 0
		ORDER BY e.created_at DESC
		LIMIT ? OFFSET ?
		`
		args = []interface{}{recipientID, limit, offset}
	}
	
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []*Email
	for rows.Next() {
		var email Email
		err := rows.Scan(
			&email.ID, &email.UUID, &email.SenderID, &email.RecipientID,
			&email.SenderEmail, &email.RecipientEmail,
			&email.Subject, &email.Body,
			&email.IsRead, &email.IsStarred, &email.IsDeleted, &email.IsDraft,
			&email.HasAttachment, &email.CreatedAt, &email.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, &email)
	}
	
	return emails, nil
}

func CountEmailsByRecipient(db *Database, recipientID int, folder string) (int, error) {
	var query string
	var args []interface{}
	
	switch folder {
	case "inbox":
		query = `SELECT COUNT(*) FROM emails WHERE recipient_id = ? AND is_deleted = 0 AND is_draft = 0`
		args = []interface{}{recipientID}
	case "sent":
		query = `SELECT COUNT(*) FROM emails WHERE sender_id = ? AND is_deleted = 0 AND is_draft = 0`
		args = []interface{}{recipientID}
	case "starred":
		query = `SELECT COUNT(*) FROM emails WHERE (sender_id = ? OR recipient_id = ?) AND is_starred = 1 AND is_deleted = 0`
		args = []interface{}{recipientID, recipientID}
	case "drafts":
		query = `SELECT COUNT(*) FROM emails WHERE sender_id = ? AND is_draft = 1 AND is_deleted = 0`
		args = []interface{}{recipientID}
	case "trash":
		query = `SELECT COUNT(*) FROM emails WHERE (sender_id = ? OR recipient_id = ?) AND is_deleted = 1`
		args = []interface{}{recipientID, recipientID}
	default:
		query = `SELECT COUNT(*) FROM emails WHERE recipient_id = ? AND is_deleted = 0 AND is_draft = 0`
		args = []interface{}{recipientID}
	}
	
	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func MarkAsRead(db *Database, emailID int) error {
	query := `UPDATE emails SET is_read = 1, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, time.Now(), emailID)
	return err
}

func ToggleStar(db *Database, emailID int) error {
	query := `UPDATE emails SET is_starred = NOT is_starred, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, time.Now(), emailID)
	return err
}

func MoveToTrash(db *Database, emailID int) error {
	query := `UPDATE emails SET is_deleted = 1, updated_at = ? WHERE id = ?`
	_, err := db.Exec(query, time.Now(), emailID)
	return err
}

func DeletePermanently(db *Database, emailID int) error {
	// 先删除附件
	_, err := db.Exec(`DELETE FROM attachments WHERE email_id = ?`, emailID)
	if err != nil {
		return err
	}
	
	// 再删除邮件
	query := `DELETE FROM emails WHERE id = ?`
	_, err = db.Exec(query, emailID)
	return err
}

func GetAllEmails(db *Database, limit, offset int) ([]*Email, error) {
	query := `
	SELECT id, uuid, sender_id, recipient_id, sender_email, recipient_email,
	       subject, body, is_read, is_starred, is_deleted, is_draft,
	       has_attachment, created_at, updated_at
	FROM emails
	ORDER BY created_at DESC
	LIMIT ? OFFSET ?
	`
	
	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var emails []*Email
	for rows.Next() {
		var email Email
		err := rows.Scan(
			&email.ID, &email.UUID, &email.SenderID, &email.RecipientID,
			&email.SenderEmail, &email.RecipientEmail,
			&email.Subject, &email.Body,
			&email.IsRead, &email.IsStarred, &email.IsDeleted, &email.IsDraft,
			&email.HasAttachment, &email.CreatedAt, &email.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, &email)
	}
	
	return emails, nil
}

func CountAllEmails(db *Database) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM emails").Scan(&count)
	return count, err
}