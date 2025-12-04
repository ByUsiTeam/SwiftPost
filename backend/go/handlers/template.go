package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"bytes"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TemplateData 用于传递数据到模板
type TemplateData struct {
	Title         string
	User          *models.User
	Message       string
	Error         string
	Data          interface{}
	RequestDomain string
	MainDomain    string
	Year          int
}

var templates *template.Template

func init() {
	// 加载模板
	loadTemplates()
}

func loadTemplates() {
	templates = template.New("").Funcs(template.FuncMap{
		"formatTime": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"timeAgo": func(t time.Time) string {
			duration := time.Since(t)
			if duration < time.Minute {
				return "刚刚"
			} else if duration < time.Hour {
				return fmt.Sprintf("%d分钟前", int(duration.Minutes()))
			} else if duration < 24*time.Hour {
				return fmt.Sprintf("%d小时前", int(duration.Hours()))
			} else if duration < 30*24*time.Hour {
				return fmt.Sprintf("%d天前", int(duration.Hours()/24))
			} else {
				return t.Format("2006-01-02")
			}
		},
		"truncate": func(s string, length int) string {
			if len(s) > length {
				return s[:length] + "..."
			}
			return s
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"subtract": func(a, b int) int {
			return a - b
		},
		"multiply": func(a, b int) int {
			return a * b
		},
		"divide": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
	})

	// 遍历模板目录
	templateDir := "frontend/templates"
	err := filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			relPath, _ := filepath.Rel(templateDir, path)
			name := strings.Replace(relPath, string(filepath.Separator), "/", -1)
			
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			
			_, err = templates.New(name).Parse(string(data))
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		utils.Error("加载模板失败: %v", err)
		// 创建默认模板
		createDefaultTemplates()
	}
	
	utils.PrintSuccess("模板加载完成")
}

func createDefaultTemplates() {
	// 创建基本模板
	baseTemplate := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - SwiftPost</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css" rel="stylesheet">
    <style>
        body { padding-top: 20px; }
        .navbar-brand { font-weight: bold; }
        .card { border-radius: 10px; }
        .unread { font-weight: bold; }
        .starred { color: #ffc107; }
    </style>
</head>
<body>
    {{template "content" .}}
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`
	
	templates.Parse(baseTemplate)
}

func renderTemplate(w http.ResponseWriter, name string, data *TemplateData) {
	// 设置默认值
	if data == nil {
		data = &TemplateData{}
	}
	if data.Title == "" {
		data.Title = "SwiftPost"
	}
	data.Year = time.Now().Year()
	
	// 执行模板
	var buf bytes.Buffer
	err := templates.ExecuteTemplate(&buf, name, data)
	if err != nil {
		utils.Error("执行模板失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(buf.Bytes())
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "安全邮件服务",
	}
	renderTemplate(w, "index.html", data)
}

func LoginPageHandler(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "用户登录",
	}
	renderTemplate(w, "login.html", data)
}

func RegisterPageHandler(w http.ResponseWriter, r *http.Request) {
	data := &TemplateData{
		Title: "用户注册",
	}
	renderTemplate(w, "register.html", data)
}

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	
	// 获取统计数据
	var unreadCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND is_read = 0 AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&unreadCount)
	if err != nil {
		unreadCount = 0
	}
	
	// 获取收件箱邮件
	emails, err := models.GetEmailsByRecipient(db, userID, 10, 0, "inbox")
	if err != nil {
		utils.Error("获取邮件失败: %v", err)
		emails = []*models.Email{}
	}
	
	// 获取发件人信息
	emailData := make([]map[string]interface{}, len(emails))
	for i, email := range emails {
		sender, _ := models.GetUserByID(db, email.SenderID)
		senderName := email.SenderEmail
		if sender != nil {
			senderName = sender.Username
		}
		
		bodyPreview := getBodyPreview(email.Body)
		if len(bodyPreview) > 100 {
			bodyPreview = bodyPreview[:100] + "..."
		}
		
		emailData[i] = map[string]interface{}{
			"id":              email.ID,
			"uuid":            email.UUID,
			"sender_name":     senderName,
			"sender_email":    email.SenderEmail,
			"subject":         email.Subject,
			"body_preview":    bodyPreview,
			"is_read":         email.IsRead,
			"is_starred":      email.IsStarred,
			"has_attachment":  email.HasAttachment,
			"created_at":      email.CreatedAt,
			"time_ago":        getTimeAgo(email.CreatedAt),
		}
	}
	
	data := &TemplateData{
		Title: "邮件仪表板",
		User:  user,
		Data: map[string]interface{}{
			"emails":       emailData,
			"unread_count": unreadCount,
			"stats": map[string]interface{}{
				"storage_used":  float64(user.StorageUsed) / (1024 * 1024),
				"storage_max":   float64(user.MaxStorage) / (1024 * 1024),
				"storage_percent": func() float64 {
					if user.MaxStorage > 0 {
						return float64(user.StorageUsed) / float64(user.MaxStorage) * 100
					}
					return 0
				}(),
			},
		},
	}
	
	renderTemplate(w, "dashboard.html", data)
}

func EmailViewHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	emailID, _ := models.GetIDFromRequest(r, "id")
	
	db := models.GetDB()
	
	// 获取邮件
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "邮件不存在", http.StatusNotFound)
			return
		}
		utils.Error("获取邮件失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	
	// 检查权限
	if email.RecipientID != userID && email.SenderID != userID {
		http.Error(w, "无权访问此邮件", http.StatusForbidden)
		return
	}
	
	// 获取用户信息
	user, _ := models.GetUserByID(db, userID)
	sender, _ := models.GetUserByID(db, email.SenderID)
	recipient, _ := models.GetUserByID(db, email.RecipientID)
	
	senderName := email.SenderEmail
	if sender != nil {
		senderName = sender.Username
	}
	
	recipientName := email.RecipientEmail
	if recipient != nil {
		recipientName = recipient.Username
	}
	
	// 如果是收件人且未读，标记为已读
	if email.RecipientID == userID && !email.IsRead {
		models.MarkAsRead(db, email.ID)
		email.IsRead = true
	}
	
	// 获取附件
	attachments, _ := models.GetAttachmentsByEmail(db, email.ID)
	
	data := &TemplateData{
		Title: email.Subject,
		User:  user,
		Data: map[string]interface{}{
			"email": map[string]interface{}{
				"id":              email.ID,
				"uuid":            email.UUID,
				"sender_id":       email.SenderID,
				"sender_email":    email.SenderEmail,
				"sender_name":     senderName,
				"recipient_id":    email.RecipientID,
				"recipient_email": email.RecipientEmail,
				"recipient_name":  recipientName,
				"subject":         email.Subject,
				"body":            template.HTML(strings.ReplaceAll(email.Body, "\n", "<br>")),
				"is_read":         email.IsRead,
				"is_starred":      email.IsStarred,
				"has_attachment":  email.HasAttachment,
				"created_at":      email.CreatedAt,
				"time_ago":        getTimeAgo(email.CreatedAt),
				"attachments":     attachments,
			},
		},
	}
	
	renderTemplate(w, "email.html", data)
}

func ProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	
	// 统计未读邮件
	var unreadCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND is_read = 0 AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&unreadCount)
	if err != nil {
		unreadCount = 0
	}
	
	// 统计已发送邮件
	var sentCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE sender_id = ? AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&sentCount)
	if err != nil {
		sentCount = 0
	}
	
	// 计算存储使用情况
	storageUsedMB := float64(user.StorageUsed) / (1024 * 1024)
	maxStorageMB := float64(user.MaxStorage) / (1024 * 1024)
	storagePercent := 0.0
	if maxStorageMB > 0 {
		storagePercent = (storageUsedMB / maxStorageMB) * 100
	}
	
	data := &TemplateData{
		Title: "个人资料",
		User:  user,
		Data: map[string]interface{}{
			"stats": map[string]interface{}{
				"unread_emails": unreadCount,
				"sent_emails":   sentCount,
				"storage": map[string]interface{}{
					"used":       storageUsedMB,
					"max":        maxStorageMB,
					"percent":    storagePercent,
					"used_bytes": user.StorageUsed,
					"max_bytes":  user.MaxStorage,
				},
			},
		},
	}
	
	renderTemplate(w, "profile.html", data)
}

func AdminHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		http.Error(w, "内部服务器错误", http.StatusInternalServerError)
		return
	}
	
	if !user.IsAdmin {
		http.Error(w, "需要管理员权限", http.StatusForbidden)
		return
	}
	
	// 获取用户统计
	userCount, err := models.CountUsers(db)
	if err != nil {
		userCount = 0
	}
	
	// 获取邮件统计
	emailCount, err := models.CountAllEmails(db)
	if err != nil {
		emailCount = 0
	}
	
	// 获取最近的用户
	recentUsers, err := models.GetAllUsers(db, 10, 0)
	if err != nil {
		recentUsers = []*models.User{}
	}
	
	// 获取系统存储使用情况
	var totalStorageUsed int64
	err = db.QueryRow("SELECT SUM(storage_used) FROM users").Scan(&totalStorageUsed)
	if err != nil {
		totalStorageUsed = 0
	}
	
	data := &TemplateData{
		Title: "管理员面板",
		User:  user,
		Data: map[string]interface{}{
			"stats": map[string]interface{}{
				"total_users":    userCount,
				"total_emails":   emailCount,
				"storage_used":   float64(totalStorageUsed) / (1024 * 1024 * 1024), // GB
				"active_users":   userCount, // 简化，实际应该查询活跃用户
			},
			"recent_users": recentUsers,
		},
	}
	
	renderTemplate(w, "admin.html", data)
}

func BlockedHandler(w http.ResponseWriter, r *http.Request) {
	config, _ := utils.LoadConfig("config.json")
	
	data := &TemplateData{
		Title:         "访问受限",
		RequestDomain: r.Host,
		MainDomain:    config.Server.Domain + ":" + config.Server.Port,
	}
	
	renderTemplate(w, "blocked.html", data)
}

func CustomDomainHandler(w http.ResponseWriter, r *http.Request) {
	// 检查是否是主域名
	config, _ := utils.LoadConfig("config.json")
	mainDomain := config.Server.Domain
	
	if r.Host == mainDomain+":"+config.Server.Port || 
	   r.Host == mainDomain || 
	   strings.HasPrefix(r.Host, "localhost") {
		// 主域名访问，返回正常页面
		path := r.URL.Path
		switch path {
		case "/":
			IndexHandler(w, r)
		case "/login":
			LoginPageHandler(w, r)
		case "/register":
			RegisterPageHandler(w, r)
		case "/dashboard":
			// 需要登录
			w.Header().Set("Location", "/login")
			w.WriteHeader(http.StatusFound)
		default:
			// API请求直接处理
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/static") {
				// 交给其他处理器
				return
			}
			http.NotFound(w, r)
		}
		return
	}
	
	// 自定义域名访问，显示阻止页面
	BlockedHandler(w, r)
}

// 辅助函数
func getBodyPreview(body string) string {
	// 移除HTML标签
	plainText := stripHTML(body)
	
	// 截取前100个字符
	if len(plainText) > 100 {
		return plainText[:100] + "..."
	}
	return plainText
}

func stripHTML(html string) string {
	var result strings.Builder
	var inTag bool
	
	for _, c := range html {
		if c == '<' {
			inTag = true
		} else if c == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(c)
		}
	}
	
	return result.String()
}

func getTimeAgo(t time.Time) string {
	duration := time.Since(t)
	
	if duration < time.Minute {
		return "刚刚"
	} else if duration < time.Hour {
		return fmt.Sprintf("%d分钟前", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%d小时前", int(duration.Hours()))
	} else if duration < 30*24*time.Hour {
		return fmt.Sprintf("%d天前", int(duration.Hours()/24))
	} else if duration < 365*24*time.Hour {
		return fmt.Sprintf("%d个月前", int(duration.Hours()/(24*30)))
	} else {
		return fmt.Sprintf("%d年前", int(duration.Hours()/(24*365)))
	}
}