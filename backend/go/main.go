package main

import (
	"SwiftPost/handlers"
	"SwiftPost/middleware"
	"SwiftPost/models"
	"SwiftPost/utils"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
)

var (
	version = "1.0.0"
	commit  = "dev"
	date    = time.Now().Format("2006-01-02")
)

func printBanner() {
	utils.PrintColored("=", 60, utils.ColorCyan)
	fmt.Println()
	utils.PrintColored("🚀 SwiftPost 邮件服务系统 v"+version, 0, utils.ColorGreen)
	fmt.Println()
	utils.PrintColored("📅 构建时间: "+date, 0, utils.ColorYellow)
	fmt.Println()
	utils.PrintColored("🏢 组织: ByUsi Team", 0, utils.ColorBlue)
	fmt.Println()
	utils.PrintColored("🌐 GitHub: github.com/byusiteam", 0, utils.ColorMagenta)
	fmt.Println()
	utils.PrintColored("=", 60, utils.ColorCyan)
	fmt.Println()
}

func startPythonService(config *utils.Config) (*exec.Cmd, error) {
	utils.PrintColored("🐍 启动 Python 数据库服务...", 0, utils.ColorYellow)
	
	// 获取当前目录
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	
	// 构建 Python 脚本路径
	var pythonScript string
	if runtime.GOOS == "windows" {
		pythonScript = wd + "\\start.py"
	} else {
		pythonScript = wd + "/start.py"
	}
	
	// 检查 Python 脚本是否存在
	if _, err := os.Stat(pythonScript); os.IsNotExist(err) {
		// 如果 start.py 不存在，使用内置的 Python 代码
		utils.PrintColored("📝 生成 Python 数据库脚本...", 0, utils.ColorYellow)
		pythonCode := `#!/usr/bin/env python3
import sys
import os
import time
import sqlite3
from pathlib import Path

def init_database():
    db_path = "data/swiftpost.db"
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    cursor.execute('''
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
    )''')
    
    cursor.execute('''
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
    )''')
    
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_emails_recipient ON emails(recipient_id, created_at DESC)')
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)')
    
    conn.commit()
    conn.close()
    return db_path

def monitor_database():
    db_path = "data/swiftpost.db"
    
    while True:
        try:
            conn = sqlite3.connect(db_path)
            cursor = conn.cursor()
            cursor.execute('SELECT 1')
            conn.close()
            time.sleep(10)
        except KeyboardInterrupt:
            print("\\n🔄 数据库监控已停止")
            break
        except Exception as e:
            print(f"❌ 数据库连接错误: {e}")
            time.sleep(5)

if __name__ == "__main__":
    print("=" * 50)
    print("🚀 SwiftPost 数据库服务")
    print("=" * 50)
    
    if len(sys.argv) > 1 and sys.argv[1] == "--child":
        print("📊 数据库服务已启动")
        db_path = init_database()
        print(f"✅ 数据库初始化完成: {db_path}")
        monitor_database()
    else:
        print("❌ 此脚本只能作为子进程启动")
        sys.exit(1)`
		
		// 写入 Python 脚本
		if err := os.WriteFile(pythonScript, []byte(pythonCode), 0755); err != nil {
			return nil, err
		}
	}
	
	// 启动 Python 进程
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("python", pythonScript, "--child")
	} else {
		cmd = exec.Command("python3", pythonScript, "--child")
	}
	
	// 设置输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// 启动进程
	if err := cmd.Start(); err != nil {
		// 尝试另一种方式
		utils.PrintColored("⚠️  尝试使用 python3 启动...", 0, utils.ColorYellow)
		if runtime.GOOS == "windows" {
			cmd = exec.Command("py", pythonScript, "--child")
		} else {
			cmd = exec.Command("python", pythonScript, "--child")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("无法启动 Python 服务: %v", err)
		}
	}
	
	// 等待一小段时间确保 Python 服务启动
	time.Sleep(2 * time.Second)
	
	utils.PrintColored("✅ Python 数据库服务已启动", 0, utils.ColorGreen)
	return cmd, nil
}

func main() {
	// 显示启动横幅
	printBanner()
	
	// 加载配置
	utils.PrintColored("📋 加载配置文件...", 0, utils.ColorYellow)
	config, err := utils.LoadConfig("config.json")
	if err != nil {
		utils.PrintColored(fmt.Sprintf("❌ 无法加载配置: %v", err), 0, utils.ColorRed)
		log.Fatal(err)
	}
	utils.PrintColored("✅ 配置加载完成", 0, utils.ColorGreen)
	
	// 创建数据目录
	if err := os.MkdirAll("data/emails", 0755); err != nil {
		utils.PrintColored(fmt.Sprintf("❌ 无法创建数据目录: %v", err), 0, utils.ColorRed)
		log.Fatal(err)
	}
	if err := os.MkdirAll("data/attachments", 0755); err != nil {
		utils.PrintColored(fmt.Sprintf("❌ 无法创建附件目录: %v", err), 0, utils.ColorRed)
		log.Fatal(err)
	}
	
	// 启动 Python 数据库服务
	var pythonCmd *exec.Cmd
	if config.Database.PythonEnabled {
		pythonCmd, err = startPythonService(config)
		if err != nil {
			utils.PrintColored(fmt.Sprintf("⚠️  Python 服务启动失败: %v", err), 0, utils.ColorYellow)
			utils.PrintColored("ℹ️  继续使用 Go 内置数据库功能", 0, utils.ColorBlue)
		} else {
			defer func() {
				if pythonCmd != nil && pythonCmd.Process != nil {
					utils.PrintColored("🔄 停止 Python 服务...", 0, utils.ColorYellow)
					pythonCmd.Process.Kill()
				}
			}()
		}
	}
	
	// 初始化数据库
	utils.PrintColored("🗄️  初始化数据库连接...", 0, utils.ColorYellow)
	db, err := models.InitDatabase(config.Database.Path)
	if err != nil {
		utils.PrintColored(fmt.Sprintf("❌ 无法初始化数据库: %v", err), 0, utils.ColorRed)
		log.Fatal(err)
	}
	defer db.Close()
	utils.PrintColored("✅ 数据库连接已建立", 0, utils.ColorGreen)
	
	// 设置第一个用户为管理员
	if config.Admin.FirstUserAdmin {
		models.SetFirstUserAsAdmin(db)
	}
	
	// 创建路由器
	router := mux.NewRouter()
	
	// 静态文件服务
	fs := http.FileServer(http.Dir("frontend/static"))
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))
	
	// WebSocket 升级器
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 在生产环境中应该更严格
		},
	}
	
	// 注册路由
	registerRoutes(router, db, upgrader)
	
	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         config.Server.Host + ":" + config.Server.Port,
		Handler:      router,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	// 启动服务器协程
	go func() {
		utils.PrintColored("🌐 启动 HTTP 服务器...", 0, utils.ColorYellow)
		utils.PrintColored(fmt.Sprintf("📡 监听地址: %s", server.Addr), 0, utils.ColorCyan)
		utils.PrintColored(fmt.Sprintf("🔗 服务域名: %s", config.Server.Domain), 0, utils.ColorCyan)
		utils.PrintColored("🚪 访问地址: http://" + server.Addr, 0, utils.ColorGreen)
		
		if config.Server.SSL.Enabled {
			utils.PrintColored("🔒 SSL/TLS 已启用", 0, utils.ColorGreen)
			if err := server.ListenAndServeTLS(
				config.Server.SSL.Cert,
				config.Server.SSL.Key,
			); err != nil && err != http.ErrServerClosed {
				utils.PrintColored(fmt.Sprintf("❌ HTTPS 服务器错误: %v", err), 0, utils.ColorRed)
				log.Fatal(err)
			}
		} else {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				utils.PrintColored(fmt.Sprintf("❌ HTTP 服务器错误: %v", err), 0, utils.ColorRed)
				log.Fatal(err)
			}
		}
	}()
	
	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	<-c
	utils.PrintColored("\n🔄 收到停止信号，正在关闭服务器...", 0, utils.ColorYellow)
	
	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		utils.PrintColored(fmt.Sprintf("❌ 服务器关闭错误: %v", err), 0, utils.ColorRed)
	}
	
	utils.PrintColored("👋 SwiftPost 服务已停止", 0, utils.ColorGreen)
	os.Exit(0)
}

func registerRoutes(router *mux.Router, db *models.Database, upgrader websocket.Upgrader) {
	// HTML 页面路由
	router.HandleFunc("/", handlers.IndexHandler).Methods("GET")
	router.HandleFunc("/login", handlers.LoginPageHandler).Methods("GET")
	router.HandleFunc("/register", handlers.RegisterPageHandler).Methods("GET")
	router.HandleFunc("/dashboard", middleware.AuthMiddleware(handlers.DashboardHandler)).Methods("GET")
	router.HandleFunc("/email/{id}", middleware.AuthMiddleware(handlers.EmailViewHandler)).Methods("GET")
	router.HandleFunc("/admin", middleware.AuthMiddleware(middleware.AdminMiddleware(handlers.AdminHandler))).Methods("GET")
	router.HandleFunc("/profile", middleware.AuthMiddleware(handlers.ProfileHandler)).Methods("GET")
	router.HandleFunc("/blocked", handlers.BlockedHandler).Methods("GET")
	
	// API 路由
	// 认证相关
	router.HandleFunc("/api/register", handlers.RegisterHandler).Methods("POST")
	router.HandleFunc("/api/login", handlers.LoginHandler).Methods("POST")
	router.HandleFunc("/api/logout", middleware.AuthMiddleware(handlers.LogoutHandler)).Methods("POST")
	router.HandleFunc("/api/refresh", handlers.RefreshTokenHandler).Methods("POST")
	
	// 用户相关
	router.HandleFunc("/api/user/profile", middleware.AuthMiddleware(handlers.GetProfileHandler)).Methods("GET")
	router.HandleFunc("/api/user/profile", middleware.AuthMiddleware(handlers.UpdateProfileHandler)).Methods("PUT")
	router.HandleFunc("/api/user/stats", middleware.AuthMiddleware(handlers.GetUserStatsHandler)).Methods("GET")
	router.HandleFunc("/api/user/domain", middleware.AuthMiddleware(handlers.UpdateDomainHandler)).Methods("PUT")
	
	// 邮件相关
	router.HandleFunc("/api/emails", middleware.AuthMiddleware(handlers.GetEmailsHandler)).Methods("GET")
	router.HandleFunc("/api/emails/send", middleware.AuthMiddleware(handlers.SendEmailHandler)).Methods("POST")
	router.HandleFunc("/api/emails/{id}", middleware.AuthMiddleware(handlers.GetEmailHandler)).Methods("GET")
	router.HandleFunc("/api/emails/{id}", middleware.AuthMiddleware(handlers.UpdateEmailHandler)).Methods("PUT")
	router.HandleFunc("/api/emails/{id}", middleware.AuthMiddleware(handlers.DeleteEmailHandler)).Methods("DELETE")
	router.HandleFunc("/api/emails/{id}/read", middleware.AuthMiddleware(handlers.MarkAsReadHandler)).Methods("PUT")
	router.HandleFunc("/api/emails/{id}/star", middleware.AuthMiddleware(handlers.ToggleStarHandler)).Methods("PUT")
	
	// 附件相关
	router.HandleFunc("/api/attachments/upload", middleware.AuthMiddleware(handlers.UploadAttachmentHandler)).Methods("POST")
	router.HandleFunc("/api/attachments/{id}/download", middleware.AuthMiddleware(handlers.DownloadAttachmentHandler)).Methods("GET")
	
	// 管理员相关
	router.HandleFunc("/api/admin/users", middleware.AuthMiddleware(middleware.AdminMiddleware(handlers.AdminGetUsersHandler))).Methods("GET")
	router.HandleFunc("/api/admin/users/{id}", middleware.AuthMiddleware(middleware.AdminMiddleware(handlers.AdminUpdateUserHandler))).Methods("PUT")
	router.HandleFunc("/api/admin/users/{id}", middleware.AuthMiddleware(middleware.AdminMiddleware(handlers.AdminDeleteUserHandler))).Methods("DELETE")
	router.HandleFunc("/api/admin/stats", middleware.AuthMiddleware(middleware.AdminMiddleware(handlers.AdminGetStatsHandler))).Methods("GET")
	router.HandleFunc("/api/admin/emails", middleware.AuthMiddleware(middleware.AdminMiddleware(handlers.AdminGetEmailsHandler))).Methods("GET")
	
	// WebSocket 路由
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handlers.WebSocketHandler(w, r, db, upgrader)
	})
	
	// 健康检查
	router.HandleFunc("/health", handlers.HealthCheckHandler).Methods("GET")
	router.HandleFunc("/api/health", handlers.HealthCheckHandler).Methods("GET")
	
	// 自定义域名处理（最后匹配）
	router.PathPrefix("/").HandlerFunc(handlers.CustomDomainHandler)
}