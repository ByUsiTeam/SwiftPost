package models

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"SwiftPost/utils"
)

type Database struct {
	*sql.DB
}

var dbInstance *Database

func InitDatabase(path string) (*Database, error) {
	// 确保目录存在
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建数据库目录: %v", err)
	}
	
	// 打开数据库连接
	sqlDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("无法打开数据库: %v", err)
	}
	
	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接失败: %v", err)
	}
	
	// 设置连接池
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(1800) // 30分钟
	
	// 创建数据库实例
	dbInstance = &Database{sqlDB}
	
	// 初始化表
	if err := initTables(dbInstance); err != nil {
		return nil, fmt.Errorf("初始化表失败: %v", err)
	}
	
	utils.PrintSuccess("数据库初始化完成")
	return dbInstance, nil
}

func initTables(db *Database) error {
	// 创建用户表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		is_admin BOOLEAN DEFAULT 0,
		custom_domain TEXT,
		storage_used INTEGER DEFAULT 0,
		max_storage INTEGER DEFAULT 1073741824,
		is_active BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)
	`)
	if err != nil {
		return fmt.Errorf("创建用户表失败: %v", err)
	}
	
	// 创建邮件表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS emails (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		uuid TEXT UNIQUE NOT NULL,
		sender_id INTEGER NOT NULL,
		recipient_id INTEGER NOT NULL,
		sender_email TEXT NOT NULL,
		recipient_email TEXT NOT NULL,
		subject TEXT NOT NULL,
		body TEXT NOT NULL,
		is_read BOOLEAN DEFAULT 0,
		is_starred BOOLEAN DEFAULT 0,
		is_deleted BOOLEAN DEFAULT 0,
		is_draft BOOLEAN DEFAULT 0,
		has_attachment BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (sender_id) REFERENCES users (id),
		FOREIGN KEY (recipient_id) REFERENCES users (id)
	)
	`)
	if err != nil {
		return fmt.Errorf("创建邮件表失败: %v", err)
	}
	
	// 创建附件表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS attachments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email_id INTEGER NOT NULL,
		uuid TEXT UNIQUE NOT NULL,
		filename TEXT NOT NULL,
		filepath TEXT NOT NULL,
		file_size INTEGER,
		mime_type TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (email_id) REFERENCES emails (id)
	)
	`)
	if err != nil {
		return fmt.Errorf("创建附件表失败: %v", err)
	}
	
	// 创建会话表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		session_token TEXT UNIQUE NOT NULL,
		ip_address TEXT,
		user_agent TEXT,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users (id)
	)
	`)
	if err != nil {
		return fmt.Errorf("创建会话表失败: %v", err)
	}
	
	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_emails_recipient ON emails(recipient_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_sender ON emails(sender_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)`,
		`CREATE INDEX IF NOT EXISTS idx_emails_uuid ON emails(uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(session_token)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_email ON attachments(email_id)`,
	}
	
	for _, index := range indexes {
		if _, err := db.Exec(index); err != nil {
			return fmt.Errorf("创建索引失败: %v", err)
		}
	}
	
	utils.PrintSuccess("数据库表创建完成")
	return nil
}

func GetDB() *Database {
	return dbInstance
}

func SetFirstUserAsAdmin(db *Database) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		utils.Error("查询用户数量失败: %v", err)
		return
	}
	
	if count == 0 {
		utils.Info("数据库中没有用户，跳过管理员设置")
		return
	}
	
	// 检查是否已有管理员
	var adminCount int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&adminCount)
	if err != nil {
		utils.Error("查询管理员数量失败: %v", err)
		return
	}
	
	if adminCount == 0 {
		// 设置第一个用户为管理员
		_, err = db.Exec("UPDATE users SET is_admin = 1 WHERE id = (SELECT MIN(id) FROM users)")
		if err != nil {
			utils.Error("设置管理员失败: %v", err)
			return
		}
		utils.PrintSuccess("已将第一个用户设置为管理员")
	}
}