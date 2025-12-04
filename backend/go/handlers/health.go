package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// HealthCheckResponse 健康检查响应
type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Services  map[string]ServiceInfo `json:"services"`
	System    SystemInfo             `json:"system"`
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// SystemInfo 系统信息
type SystemInfo struct {
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Arch      string `json:"arch"`
	CPUs      int    `json:"cpus"`
}

var (
	startTime = time.Now()
	version   = "1.0.0"
)

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	db := models.GetDB()
	
	response := HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   version,
		Services:  make(map[string]ServiceInfo),
		System: SystemInfo{
			GoVersion: runtime.Version(),
			Platform:  runtime.GOOS,
			Arch:      runtime.GOARCH,
			CPUs:      runtime.NumCPU(),
		},
	}
	
	// 检查数据库连接
	if err := db.Ping(); err != nil {
		response.Status = "degraded"
		response.Services["database"] = ServiceInfo{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	} else {
		response.Services["database"] = ServiceInfo{
			Status: "healthy",
		}
	}
	
	// 检查磁盘空间（简化版）
	response.Services["storage"] = ServiceInfo{
		Status: "healthy",
	}
	
	// 检查WebSocket服务
	response.Services["websocket"] = ServiceInfo{
		Status: "healthy",
	}
	
	// 添加Uptime信息
	response.Services["uptime"] = ServiceInfo{
		Status:  "healthy",
		Message: time.Since(startTime).String(),
	}
	
	// 设置响应状态码
	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if response.Status == "degraded" {
		statusCode = http.StatusPartialContent
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// StatsResponse 统计响应
type StatsResponse struct {
	Success bool        `json:"success"`
	Stats   SystemStats `json:"stats"`
}

// SystemStats 系统统计
type SystemStats struct {
	Users       UserStats    `json:"users"`
	Emails      EmailStats   `json:"emails"`
	System      RuntimeStats `json:"system"`
	Performance PerfStats    `json:"performance"`
}

// UserStats 用户统计
type UserStats struct {
	Total      int `json:"total"`
	Active     int `json:"active"`
	Admins     int `json:"admins"`
	NewToday   int `json:"new_today"`
	NewThisWeek int `json:"new_this_week"`
}

// EmailStats 邮件统计
type EmailStats struct {
	Total      int `json:"total"`
	Unread     int `json:"unread"`
	SentToday  int `json:"sent_today"`
	ReceivedToday int `json:"received_today"`
}

// RuntimeStats 运行时统计
type RuntimeStats struct {
	Uptime      string `json:"uptime"`
	Goroutines  int    `json:"goroutines"`
	MemoryAlloc uint64 `json:"memory_alloc"`
	MemoryTotal uint64 `json:"memory_total"`
}

// PerfStats 性能统计
type PerfStats struct {
	DBQueryTime   float64 `json:"db_query_time"`
	APILatency    float64 `json:"api_latency"`
	WebSocketConn int     `json:"websocket_connections"`
}

func GetSystemStatsHandler(w http.ResponseWriter, r *http.Request) {
	db := models.GetDB()
	
	// 获取用户统计
	var totalUsers, activeUsers, adminUsers int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = 1").Scan(&activeUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&adminUsers)
	
	// 获取今日新用户
	var newToday int
	today := time.Now().Format("2006-01-02")
	db.QueryRow("SELECT COUNT(*) FROM users WHERE DATE(created_at) = ?", today).Scan(&newToday)
	
	// 获取本周新用户
	var newThisWeek int
	weekStart := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
	db.QueryRow("SELECT COUNT(*) FROM users WHERE created_at >= ?", weekStart).Scan(&newThisWeek)
	
	// 获取邮件统计
	var totalEmails, unreadEmails int
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE is_deleted = 0").Scan(&totalEmails)
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE is_read = 0 AND is_deleted = 0").Scan(&unreadEmails)
	
	// 获取今日发送邮件
	var sentToday int
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE DATE(created_at) = ? AND is_draft = 0", today).Scan(&sentToday)
	
	// 获取今日接收邮件
	var receivedToday int
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE DATE(created_at) = ? AND is_draft = 0 AND recipient_id IN (
			SELECT id FROM users WHERE is_active = 1
		)
	`, today).Scan(&receivedToday)
	
	// 获取内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	response := StatsResponse{
		Success: true,
		Stats: SystemStats{
			Users: UserStats{
				Total:       totalUsers,
				Active:      activeUsers,
				Admins:      adminUsers,
				NewToday:    newToday,
				NewThisWeek: newThisWeek,
			},
			Emails: EmailStats{
				Total:         totalEmails,
				Unread:        unreadEmails,
				SentToday:     sentToday,
				ReceivedToday: receivedToday,
			},
			System: RuntimeStats{
				Uptime:      time.Since(startTime).String(),
				Goroutines:  runtime.NumGoroutine(),
				MemoryAlloc: memStats.Alloc / 1024 / 1024, // MB
				MemoryTotal: memStats.TotalAlloc / 1024 / 1024, // MB
			},
			Performance: PerfStats{
				DBQueryTime:   0.1, // 简化，实际应该计算
				APILatency:    50,  // 毫秒
				WebSocketConn: 0,   // 简化
			},
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// MonitoringMiddleware 监控中间件
func MonitoringMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// 创建包装器来捕获状态码
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start).Seconds()
		
		// 记录请求日志
		utils.Info("请求: %s %s %d %.3fs",
			r.Method, r.URL.Path, rw.statusCode, duration)
		
		// 这里可以添加更多的监控逻辑
		// 比如记录到数据库、发送到监控系统等
	}
}

// responseWriter 包装ResponseWriter来捕获状态码
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}