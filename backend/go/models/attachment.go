package models

import (
	"database/sql"
	"time"
)

type Attachment struct {
	ID        int       `json:"id"`
	EmailID   int       `json:"email_id"`
	UUID      string    `json:"uuid"`
	Filename  string    `json:"filename"`
	Filepath  string    `json:"filepath"`
	FileSize  int64     `json:"file_size"`
	MimeType  string    `json:"mime_type"`
	CreatedAt time.Time `json:"created_at"`
}

func CreateAttachment(db *Database, attachment *Attachment) (int64, error) {
	query := `
	INSERT INTO attachments (email_id, uuid, filename, filepath, file_size, mime_type, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	result, err := db.Exec(query,
		attachment.EmailID, attachment.UUID, attachment.Filename,
		attachment.Filepath, attachment.FileSize, attachment.MimeType,
		time.Now(),
	)
	
	if err != nil {
		return 0, err
	}
	
	return result.LastInsertId()
}

func GetAttachmentByID(db *Database, id int) (*Attachment, error) {
	var attachment Attachment
	query := `
	SELECT id, email_id, uuid, filename, filepath, file_size, mime_type, created_at
	FROM attachments WHERE id = ?
	`
	
	err := db.QueryRow(query, id).Scan(
		&attachment.ID, &attachment.EmailID, &attachment.UUID,
		&attachment.Filename, &attachment.Filepath, &attachment.FileSize,
		&attachment.MimeType, &attachment.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &attachment, nil
}

func GetAttachmentByUUID(db *Database, uuid string) (*Attachment, error) {
	var attachment Attachment
	query := `
	SELECT id, email_id, uuid, filename, filepath, file_size, mime_type, created_at
	FROM attachments WHERE uuid = ?
	`
	
	err := db.QueryRow(query, uuid).Scan(
		&attachment.ID, &attachment.EmailID, &attachment.UUID,
		&attachment.Filename, &attachment.Filepath, &attachment.FileSize,
		&attachment.MimeType, &attachment.CreatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &attachment, nil
}

func GetAttachmentsByEmail(db *Database, emailID int) ([]*Attachment, error) {
	query := `
	SELECT id, email_id, uuid, filename, filepath, file_size, mime_type, created_at
	FROM attachments WHERE email_id = ?
	ORDER BY created_at DESC
	`
	
	rows, err := db.Query(query, emailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var attachments []*Attachment
	for rows.Next() {
		var attachment Attachment
		err := rows.Scan(
			&attachment.ID, &attachment.EmailID, &attachment.UUID,
			&attachment.Filename, &attachment.Filepath, &attachment.FileSize,
			&attachment.MimeType, &attachment.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, &attachment)
	}
	
	return attachments, nil
}

func DeleteAttachment(db *Database, id int) error {
	query := `DELETE FROM attachments WHERE id = ?`
	_, err := db.Exec(query, id)
	return err
}

func CountAttachmentsByUser(db *Database, userID int) (int, error) {
	query := `
	SELECT COUNT(*) FROM attachments a
	JOIN emails e ON a.email_id = e.id
	WHERE e.sender_id = ? OR e.recipient_id = ?
	`
	
	var count int
	err := db.QueryRow(query, userID, userID).Scan(&count)
	return count, err
}

func GetTotalAttachmentSizeByUser(db *Database, userID int) (int64, error) {
	query := `
	SELECT COALESCE(SUM(a.file_size), 0) FROM attachments a
	JOIN emails e ON a.email_id = e.id
	WHERE e.sender_id = ? OR e.recipient_id = ?
	`
	
	var totalSize int64
	err := db.QueryRow(query, userID, userID).Scan(&totalSize)
	return totalSize, err
}