package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

// AdminGetUsersHandler 获取用户列表
func AdminGetUsersHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 验证管理员权限
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil || !user.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	// 获取分页参数
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	offset := (page - 1) * limit
	
	// 获取用户列表
	users, err := models.GetAllUsers(db, limit, offset)
	if err != nil {
		utils.Error("获取用户列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取用户列表失败",
		})
		return
	}
	
	// 获取用户总数
	total, err := models.CountUsers(db)
	if err != nil {
		utils.Error("统计用户数量失败: %v", err)
		total = len(users)
	}
	
	// 准备响应数据
	userList := make([]map[string]interface{}, len(users))
	for i, user := range users {
		// 获取用户的邮件统计
		var sentCount, receivedCount int
		db.QueryRow("SELECT COUNT(*) FROM emails WHERE sender_id = ? AND is_deleted = 0", user.ID).Scan(&sentCount)
		db.QueryRow("SELECT COUNT(*) FROM emails WHERE recipient_id = ? AND is_deleted = 0", user.ID).Scan(&receivedCount)
		
		userList[i] = map[string]interface{}{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"is_admin":      user.IsAdmin,
			"custom_domain": user.CustomDomain,
			"is_active":     user.IsActive,
			"storage": map[string]interface{}{
				"used":  float64(user.StorageUsed) / (1024 * 1024),
				"max":   float64(user.MaxStorage) / (1024 * 1024),
				"percent": func() float64 {
					if user.MaxStorage > 0 {
						return float64(user.StorageUsed) / float64(user.MaxStorage) * 100
					}
					return 0
				}(),
			},
			"stats": map[string]interface{}{
				"sent_emails":     sentCount,
				"received_emails": receivedCount,
			},
			"created_at": user.CreatedAt.Format("2006-01-02 15:04:05"),
			"updated_at": user.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"users":   userList,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + limit - 1) / limit,
		},
	})
}

// AdminUpdateUserHandler 更新用户信息
func AdminUpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}
	
	// 验证管理员权限
	db := models.GetDB()
	admin, err := models.GetUserByID(db, adminID)
	if err != nil || !admin.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	// 不能修改自己（通过这个接口）
	if adminID == userID {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "不能修改自己的管理员状态",
		})
		return
	}
	
	var updateData struct {
		Username     *string `json:"username"`
		Email        *string `json:"email"`
		IsAdmin      *bool   `json:"is_admin"`
		IsActive     *bool   `json:"is_active"`
		CustomDomain *string `json:"custom_domain"`
		MaxStorage   *int64  `json:"max_storage"`
		Password     *string `json:"password"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	// 获取用户信息
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "用户不存在",
			})
			return
		}
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取用户信息失败",
		})
		return
	}
	
	// 更新字段
	updated := false
	
	if updateData.Username != nil && *updateData.Username != user.Username {
		// 检查用户名是否已存在
		existingUser, err := models.GetUserByUsername(db, *updateData.Username)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			respondJSON(w, http.StatusConflict, map[string]interface{}{
				"success": false,
				"message": "用户名已被使用",
			})
			return
		}
		user.Username = *updateData.Username
		updated = true
	}
	
	if updateData.Email != nil && *updateData.Email != user.Email {
		// 检查邮箱是否已存在
		existingUser, err := models.GetUserByEmail(db, *updateData.Email)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			respondJSON(w, http.StatusConflict, map[string]interface{}{
				"success": false,
				"message": "邮箱已被注册",
			})
			return
		}
		user.Email = *updateData.Email
		updated = true
	}
	
	if updateData.IsAdmin != nil && *updateData.IsAdmin != user.IsAdmin {
		user.IsAdmin = *updateData.IsAdmin
		updated = true
	}
	
	if updateData.IsActive != nil && *updateData.IsActive != user.IsActive {
		user.IsActive = *updateData.IsActive
		updated = true
	}
	
	if updateData.CustomDomain != nil && *updateData.CustomDomain != user.CustomDomain {
		user.CustomDomain = *updateData.CustomDomain
		updated = true
	}
	
	if updateData.MaxStorage != nil && *updateData.MaxStorage != user.MaxStorage {
		user.MaxStorage = *updateData.MaxStorage
		updated = true
	}
	
	if updateData.Password != nil && *updateData.Password != "" {
		// 哈希新密码
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*updateData.Password), bcrypt.DefaultCost)
		if err != nil {
			utils.Error("密码哈希失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "密码处理失败",
			})
			return
		}
		user.PasswordHash = string(hashedPassword)
		updated = true
	}
	
	if updated {
		if err := models.UpdateUser(db, user); err != nil {
			utils.Error("更新用户信息失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "更新用户信息失败",
			})
			return
		}
		
		utils.Info("管理员 %d 更新了用户 %d 的信息", adminID, userID)
		
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "用户信息更新成功",
			"user": map[string]interface{}{
				"id":            user.ID,
				"username":      user.Username,
				"email":         user.Email,
				"is_admin":      user.IsAdmin,
				"is_active":     user.IsActive,
				"custom_domain": user.CustomDomain,
				"max_storage":   user.MaxStorage,
			},
		})
	} else {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": "没有需要更新的信息",
		})
	}
}

// AdminDeleteUserHandler 删除用户
func AdminDeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}
	
	// 验证管理员权限
	db := models.GetDB()
	admin, err := models.GetUserByID(db, adminID)
	if err != nil || !admin.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	// 不能删除自己
	if adminID == userID {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "不能删除自己",
		})
		return
	}
	
	// 检查用户是否存在
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "用户不存在",
			})
			return
		}
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取用户信息失败",
		})
		return
	}
	
	// 永久删除用户的所有数据
	// 注意：这是一个危险操作，实际生产中应该使用软删除
	// 这里为了简化，直接硬删除
	
	// 删除用户的邮件和附件
	// 先获取用户的所有邮件
	rows, err := db.Query("SELECT id FROM emails WHERE sender_id = ? OR recipient_id = ?", userID, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var emailID int
			if err := rows.Scan(&emailID); err == nil {
				// 删除邮件的附件
				db.Exec("DELETE FROM attachments WHERE email_id = ?", emailID)
			}
		}
	}
	
	// 删除用户的邮件
	db.Exec("DELETE FROM emails WHERE sender_id = ? OR recipient_id = ?", userID, userID)
	
	// 删除用户的会话
	db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	
	// 删除用户
	if err := models.DeleteUser(db, userID); err != nil {
		utils.Error("删除用户失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "删除用户失败",
		})
		return
	}
	
	utils.Info("管理员 %d 删除了用户 %d (%s)", adminID, userID, user.Email)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "用户删除成功",
	})
}

// AdminGetStatsHandler 获取系统统计信息
func AdminGetStatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 验证管理员权限
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil || !user.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	// 获取用户统计
	var totalUsers, activeUsers, adminUsers int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = 1").Scan(&activeUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_admin = 1").Scan(&adminUsers)
	
	// 获取邮件统计
	var totalEmails, unreadEmails, todayEmails int
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE is_deleted = 0").Scan(&totalEmails)
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE is_read = 0 AND is_deleted = 0").Scan(&unreadEmails)
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE DATE(created_at) = DATE('now') AND is_deleted = 0").Scan(&todayEmails)
	
	// 获取存储统计
	var totalStorageUsed, totalStorageCapacity int64
	db.QueryRow("SELECT SUM(storage_used) FROM users").Scan(&totalStorageUsed)
	db.QueryRow("SELECT SUM(max_storage) FROM users").Scan(&totalStorageCapacity)
	
	// 获取今日新用户
	var newUsersToday int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE DATE(created_at) = DATE('now')").Scan(&newUsersToday)
	
	// 获取最近7天活跃用户
	var activeUsers7Days int
	db.QueryRow(`
		SELECT COUNT(DISTINCT user_id) FROM (
			SELECT sender_id as user_id FROM emails WHERE created_at >= DATE('now', '-7 days')
			UNION
			SELECT recipient_id as user_id FROM emails WHERE created_at >= DATE('now', '-7 days')
		)`, 
	).Scan(&activeUsers7Days)
	
	// 获取附件统计
	var totalAttachments, attachmentSize int64
	db.QueryRow("SELECT COUNT(*) FROM attachments").Scan(&totalAttachments)
	db.QueryRow("SELECT COALESCE(SUM(file_size), 0) FROM attachments").Scan(&attachmentSize)
	
	// 获取系统信息
	config, _ := utils.LoadConfig("config.json")
	
	stats := map[string]interface{}{
		"users": map[string]interface{}{
			"total":          totalUsers,
			"active":         activeUsers,
			"admins":         adminUsers,
			"new_today":      newUsersToday,
			"active_7_days":  activeUsers7Days,
			"inactive":       totalUsers - activeUsers,
		},
		"emails": map[string]interface{}{
			"total":          totalEmails,
			"unread":         unreadEmails,
			"today":          todayEmails,
			"avg_per_user":   float64(totalEmails) / float64(totalUsers),
		},
		"storage": map[string]interface{}{
			"used":           float64(totalStorageUsed) / (1024 * 1024 * 1024), // GB
			"capacity":       float64(totalStorageCapacity) / (1024 * 1024 * 1024), // GB
			"usage_percent": func() float64 {
				if totalStorageCapacity > 0 {
					return float64(totalStorageUsed) / float64(totalStorageCapacity) * 100
				}
				return 0
			}(),
			"attachments": map[string]interface{}{
				"count":        totalAttachments,
				"size":         float64(attachmentSize) / (1024 * 1024), // MB
			},
		},
		"system": map[string]interface{}{
			"domain":         config.Server.Domain,
			"port":           config.Server.Port,
			"ssl_enabled":    config.Server.SSL.Enabled,
			"max_email_size": config.Email.MaxEmailSize,
			"websocket":      config.WebSocket.Enabled,
		},
		"performance": map[string]interface{}{
			"db_connections": 25, // SQLite默认连接数
			"rate_limit":     config.Security.RateLimit,
			"token_expiry":   config.Security.TokenExpiry,
		},
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// AdminGetEmailsHandler 获取所有邮件（管理员）
func AdminGetEmailsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 验证管理员权限
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil || !user.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	// 获取分页参数
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	offset := (page - 1) * limit
	
	// 获取搜索参数
	search := r.URL.Query().Get("search")
	
	var emails []*models.Email
	var total int
	
	if search != "" {
		// 搜索邮件
		query := `
			SELECT e.id, e.uuid, e.sender_id, e.recipient_id, e.sender_email, e.recipient_email,
			       e.subject, e.body, e.is_read, e.is_starred, e.is_deleted, e.is_draft,
			       e.has_attachment, e.created_at, e.updated_at
			FROM emails e
			WHERE (e.subject LIKE ? OR e.body LIKE ? OR e.sender_email LIKE ? OR e.recipient_email LIKE ?)
			  AND e.is_deleted = 0
			ORDER BY e.created_at DESC
			LIMIT ? OFFSET ?
		`
		
		searchPattern := "%" + search + "%"
		rows, err := db.Query(query, searchPattern, searchPattern, searchPattern, searchPattern, limit, offset)
		if err != nil {
			utils.Error("搜索邮件失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "搜索邮件失败",
			})
			return
		}
		defer rows.Close()
		
		emails = make([]*models.Email, 0)
		for rows.Next() {
			var email models.Email
			err := rows.Scan(
				&email.ID, &email.UUID, &email.SenderID, &email.RecipientID,
				&email.SenderEmail, &email.RecipientEmail,
				&email.Subject, &email.Body,
				&email.IsRead, &email.IsStarred, &email.IsDeleted, &email.IsDraft,
				&email.HasAttachment, &email.CreatedAt, &email.UpdatedAt,
			)
			if err != nil {
				continue
			}
			emails = append(emails, &email)
		}
		
		// 获取总数
		db.QueryRow(`
			SELECT COUNT(*) FROM emails 
			WHERE (subject LIKE ? OR body LIKE ? OR sender_email LIKE ? OR recipient_email LIKE ?)
			  AND is_deleted = 0
		`, searchPattern, searchPattern, searchPattern, searchPattern).Scan(&total)
		
	} else {
		// 获取所有邮件
		var err error
		emails, err = models.GetAllEmails(db, limit, offset)
		if err != nil {
			utils.Error("获取邮件列表失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "获取邮件列表失败",
			})
			return
		}
		
		total, err = models.CountAllEmails(db)
		if err != nil {
			total = len(emails)
		}
	}
	
	// 获取发件人和收件人信息
	emailList := make([]map[string]interface{}, len(emails))
	for i, email := range emails {
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
		
		bodyPreview := getBodyPreview(email.Body)
		if len(bodyPreview) > 100 {
			bodyPreview = bodyPreview[:100] + "..."
		}
		
		emailList[i] = map[string]interface{}{
			"id":              email.ID,
			"uuid":            email.UUID,
			"sender_id":       email.SenderID,
			"sender_email":    email.SenderEmail,
			"sender_name":     senderName,
			"recipient_id":    email.RecipientID,
			"recipient_email": email.RecipientEmail,
			"recipient_name":  recipientName,
			"subject":         email.Subject,
			"body_preview":    bodyPreview,
			"is_read":         email.IsRead,
			"is_starred":      email.IsStarred,
			"is_deleted":      email.IsDeleted,
			"is_draft":        email.IsDraft,
			"has_attachment":  email.HasAttachment,
			"created_at":      email.CreatedAt.Format("2006-01-02 15:04:05"),
			"time_ago":        getTimeAgo(email.CreatedAt),
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"emails":  emailList,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + limit - 1) / limit,
		},
		"search": search,
	})
}

// AdminCreateUserHandler 创建新用户（管理员）
func AdminCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	adminID := r.Context().Value("user_id").(int)
	
	// 验证管理员权限
	db := models.GetDB()
	admin, err := models.GetUserByID(db, adminID)
	if err != nil || !admin.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	var req struct {
		Username     string `json:"username"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		IsAdmin      bool   `json:"is_admin"`
		IsActive     bool   `json:"is_active"`
		CustomDomain string `json:"custom_domain"`
		MaxStorage   int64  `json:"max_storage"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	// 验证输入
	if strings.TrimSpace(req.Username) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "用户名不能为空",
		})
		return
	}
	
	if strings.TrimSpace(req.Email) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "邮箱不能为空",
		})
		return
	}
	
	if strings.TrimSpace(req.Password) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "密码不能为空",
		})
		return
	}
	
	if len(req.Password) < 6 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "密码长度至少6位",
		})
		return
	}
	
	// 检查用户名和邮箱是否已存在
	existingUser, _ := models.GetUserByUsername(db, req.Username)
	if existingUser != nil {
		respondJSON(w, http.StatusConflict, map[string]interface{}{
			"success": false,
			"message": "用户名已被使用",
		})
		return
	}
	
	existingUser, _ = models.GetUserByEmail(db, req.Email)
	if existingUser != nil {
		respondJSON(w, http.StatusConflict, map[string]interface{}{
			"success": false,
			"message": "邮箱已被注册",
		})
		return
	}
	
	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error("密码哈希失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "密码处理失败",
		})
		return
	}
	
	// 创建用户
	userID, err := models.CreateUser(db, req.Username, req.Email, string(hashedPassword))
	if err != nil {
		utils.Error("创建用户失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "创建用户失败",
		})
		return
	}
	
	// 更新额外信息
	user, err := models.GetUserByID(db, int(userID))
	if err != nil {
		utils.Error("获取新用户信息失败: %v", err)
	} else {
		user.IsAdmin = req.IsAdmin
		user.IsActive = req.IsActive
		user.CustomDomain = req.CustomDomain
		user.MaxStorage = req.MaxStorage
		
		if err := models.UpdateUser(db, user); err != nil {
			utils.Error("更新用户信息失败: %v", err)
		}
	}
	
	utils.Info("管理员 %d 创建了新用户 %d (%s)", adminID, userID, req.Email)
	
	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "用户创建成功",
		"user": map[string]interface{}{
			"id":            userID,
			"username":      req.Username,
			"email":         req.Email,
			"is_admin":      req.IsAdmin,
			"is_active":     req.IsActive,
			"custom_domain": req.CustomDomain,
			"max_storage":   req.MaxStorage,
		},
	})
}

// AdminGetSystemLogsHandler 获取系统日志
func AdminGetSystemLogsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 验证管理员权限
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil || !user.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	// 获取日志文件（简化版）
	logs := []string{
		"系统启动成功",
		"数据库连接已建立",
		"WebSocket服务已启动",
		time.Now().Format("2006-01-02 15:04:05") + " - 管理员访问日志",
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"logs":    logs,
	})
}

// AdminSendSystemNotificationHandler 发送系统通知
func AdminSendSystemNotificationHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 验证管理员权限
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil || !user.IsAdmin {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "需要管理员权限",
		})
		return
	}
	
	var req struct {
		Title   string `json:"title"`
		Message string `json:"message"`
		Type    string `json:"type"` // info, warning, error
		ToAll   bool   `json:"to_all"`
		UserIDs []int  `json:"user_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	if req.Title == "" || req.Message == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "标题和内容不能为空",
		})
		return
	}
	
	// 创建WebSocket通知消息
	notification := WebSocketMessage{
		Type: "system_notification",
		Payload: map[string]interface{}{
			"title":   req.Title,
			"message": req.Message,
			"type":    req.Type,
			"from":    "系统管理员",
			"time":    time.Now().Format("2006-01-02 15:04:05"),
		},
		Timestamp: time.Now(),
	}
	
	// 发送通知
	if req.ToAll {
		// 发送给所有在线用户
		manager.Broadcast <- notification
		utils.Info("管理员 %d 发送了系统通知给所有用户", userID)
	} else if len(req.UserIDs) > 0 {
		// 发送给指定用户
		for _, targetUserID := range req.UserIDs {
			SendToUser(targetUserID, notification)
		}
		utils.Info("管理员 %d 发送了系统通知给用户 %v", userID, req.UserIDs)
	} else {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "必须指定接收用户",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "系统通知发送成功",
	})
}

// 需要导入的包
import "time"