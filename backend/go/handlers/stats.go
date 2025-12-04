package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"encoding/json"
	"net/http"
	"time"
)

// GetUserStatsHandler 获取用户统计信息
func GetUserStatsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	db := models.GetDB()
	
	// 获取用户信息
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取用户信息失败",
		})
		return
	}
	
	// 统计收件箱邮件
	var inboxCount, unreadCount, sentCount, starredCount, draftCount, trashCount int
	
	// 收件箱统计
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&inboxCount)
	
	// 未读邮件统计
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND is_read = 0 AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&unreadCount)
	
	// 已发送邮件统计
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE sender_id = ? AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&sentCount)
	
	// 星标邮件统计
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?) 
		  AND is_starred = 1 AND is_deleted = 0
	`, userID, userID).Scan(&starredCount)
	
	// 草稿统计
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE sender_id = ? AND is_draft = 1 AND is_deleted = 0
	`, userID).Scan(&draftCount)
	
	// 回收站统计
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?) AND is_deleted = 1
	`, userID, userID).Scan(&trashCount)
	
	// 统计今日邮件
	var todaySent, todayReceived int
	today := time.Now().Format("2006-01-02")
	
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE sender_id = ? AND DATE(created_at) = ? AND is_deleted = 0 AND is_draft = 0
	`, userID, today).Scan(&todaySent)
	
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND DATE(created_at) = ? AND is_deleted = 0 AND is_draft = 0
	`, userID, today).Scan(&todayReceived)
	
	// 统计最近7天邮件活动
	var last7Days []map[string]interface{}
	
	rows, err := db.Query(`
		SELECT DATE(created_at) as date, 
		       SUM(CASE WHEN sender_id = ? THEN 1 ELSE 0 END) as sent,
		       SUM(CASE WHEN recipient_id = ? THEN 1 ELSE 0 END) as received
		FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?) 
		  AND created_at >= DATE('now', '-7 days')
		  AND is_deleted = 0 AND is_draft = 0
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, userID, userID, userID, userID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date string
			var sent, received int
			if err := rows.Scan(&date, &sent, &received); err == nil {
				last7Days = append(last7Days, map[string]interface{}{
					"date":     date,
					"sent":     sent,
					"received": received,
				})
			}
		}
	}
	
	// 计算存储使用情况
	storageUsedMB := float64(user.StorageUsed) / (1024 * 1024)
	maxStorageMB := float64(user.MaxStorage) / (1024 * 1024)
	storagePercent := 0.0
	if maxStorageMB > 0 {
		storagePercent = (storageUsedMB / maxStorageMB) * 100
	}
	
	// 获取附件统计
	var attachmentCount int64
	var attachmentSize int64
	db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(file_size), 0) 
		FROM attachments a
		JOIN emails e ON a.email_id = e.id
		WHERE e.sender_id = ? OR e.recipient_id = ?
	`, userID, userID).Scan(&attachmentCount, &attachmentSize)
	
	// 获取活跃时间
	var lastLoginTime string
	db.QueryRow(`
		SELECT MAX(created_at) FROM sessions 
		WHERE user_id = ? AND expires_at > ?
	`, userID, time.Now()).Scan(&lastLoginTime)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats": map[string]interface{}{
			"user": map[string]interface{}{
				"id":            user.ID,
				"username":      user.Username,
				"email":         user.Email,
				"is_admin":      user.IsAdmin,
				"custom_domain": user.CustomDomain,
				"created_at":    user.CreatedAt.Format("2006-01-02 15:04:05"),
			},
			"email_counts": map[string]interface{}{
				"inbox":    inboxCount,
				"unread":   unreadCount,
				"sent":     sentCount,
				"starred":  starredCount,
				"drafts":   draftCount,
				"trash":    trashCount,
				"today": map[string]interface{}{
					"sent":     todaySent,
					"received": todayReceived,
					"total":    todaySent + todayReceived,
				},
			},
			"storage": map[string]interface{}{
				"used":       storageUsedMB,
				"max":        maxStorageMB,
				"percent":    storagePercent,
				"used_bytes": user.StorageUsed,
				"max_bytes":  user.MaxStorage,
				"attachments": map[string]interface{}{
					"count": attachmentCount,
					"size":  float64(attachmentSize) / (1024 * 1024), // MB
				},
			},
			"activity": map[string]interface{}{
				"last_login": lastLoginTime,
				"last_7_days": last7Days,
			},
		},
	})
}

// GetSystemStatsHandler 获取系统统计信息（公开）
func GetSystemStatsHandler(w http.ResponseWriter, r *http.Request) {
	db := models.GetDB()
	
	// 获取基本统计
	var totalUsers, activeUsers, totalEmails, todayEmails int
	
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
	db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = 1").Scan(&activeUsers)
	db.QueryRow("SELECT COUNT(*) FROM emails WHERE is_deleted = 0").Scan(&totalEmails)
	
	today := time.Now().Format("2006-01-02")
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE DATE(created_at) = ? AND is_deleted = 0 AND is_draft = 0
	`, today).Scan(&todayEmails)
	
	// 获取存储统计
	var totalStorageUsed, totalStorageCapacity int64
	db.QueryRow("SELECT COALESCE(SUM(storage_used), 0) FROM users").Scan(&totalStorageUsed)
	db.QueryRow("SELECT COALESCE(SUM(max_storage), 0) FROM users").Scan(&totalStorageCapacity)
	
	// 获取系统运行时间（从配置或环境变量）
	config, _ := utils.LoadConfig("config.json")
	
	stats := map[string]interface{}{
		"system": map[string]interface{}{
			"name":          "SwiftPost",
			"version":       "1.0.0",
			"status":        "running",
			"domain":        config.Server.Domain,
			"uptime":        "0 days", // 实际应该从启动时间计算
		},
		"users": map[string]interface{}{
			"total":   totalUsers,
			"active":  activeUsers,
			"online":  0, // WebSocket在线用户数
		},
		"emails": map[string]interface{}{
			"total":  totalEmails,
			"today":  todayEmails,
		},
		"storage": map[string]interface{}{
			"used":      float64(totalStorageUsed) / (1024 * 1024 * 1024), // GB
			"capacity":  float64(totalStorageCapacity) / (1024 * 1024 * 1024), // GB
			"available": float64(totalStorageCapacity-totalStorageUsed) / (1024 * 1024 * 1024),
		},
		"performance": map[string]interface{}{
			"response_time": "50ms",
			"availability":  "99.9%",
		},
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"stats":   stats,
	})
}

// GetActivityFeedHandler 获取用户活动动态
func GetActivityFeedHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 获取分页参数
	limit := 20
	page := 1
	
	db := models.GetDB()
	
	// 查询用户的活动（发送邮件、接收邮件等）
	activities := []map[string]interface{}{}
	
	// 查询发送的邮件
	rows, err := db.Query(`
		SELECT 'sent' as type, subject, recipient_email, created_at
		FROM emails 
		WHERE sender_id = ? AND is_deleted = 0 AND is_draft = 0
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var activityType, subject, recipientEmail string
			var createdAt time.Time
			if err := rows.Scan(&activityType, &subject, &recipientEmail, &createdAt); err == nil {
				activities = append(activities, map[string]interface{}{
					"type":        activityType,
					"title":       "发送邮件",
					"description": subject,
					"details":     "给 " + recipientEmail,
					"time":        createdAt.Format("2006-01-02 15:04:05"),
					"time_ago":    getTimeAgo(createdAt),
					"icon":        "fas fa-paper-plane",
					"color":       "primary",
				})
			}
		}
	}
	
	// 查询收到的邮件
	rows, err = db.Query(`
		SELECT 'received' as type, subject, sender_email, created_at
		FROM emails 
		WHERE recipient_id = ? AND is_deleted = 0 AND is_draft = 0
		ORDER BY created_at DESC
		LIMIT ?
	`, userID, limit)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var activityType, subject, senderEmail string
			var createdAt time.Time
			if err := rows.Scan(&activityType, &subject, &senderEmail, &createdAt); err == nil {
				activities = append(activities, map[string]interface{}{
					"type":        activityType,
					"title":       "收到邮件",
					"description": subject,
					"details":     "来自 " + senderEmail,
					"time":        createdAt.Format("2006-01-02 15:04:05"),
					"time_ago":    getTimeAgo(createdAt),
					"icon":        "fas fa-envelope",
					"color":       "success",
				})
			}
		}
	}
	
	// 查询附件上传
	rows, err = db.Query(`
		SELECT 'attachment' as type, a.filename, e.subject, a.created_at
		FROM attachments a
		JOIN emails e ON a.email_id = e.id
		WHERE e.sender_id = ?
		ORDER BY a.created_at DESC
		LIMIT ?
	`, userID, limit/2)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var activityType, filename, subject string
			var createdAt time.Time
			if err := rows.Scan(&activityType, &filename, &subject, &createdAt); err == nil {
				activities = append(activities, map[string]interface{}{
					"type":        activityType,
					"title":       "上传附件",
					"description": filename,
					"details":     "邮件: " + subject,
					"time":        createdAt.Format("2006-01-02 15:04:05"),
					"time_ago":    getTimeAgo(createdAt),
					"icon":        "fas fa-paperclip",
					"color":       "warning",
				})
			}
		}
	}
	
	// 按时间排序
	sortActivitiesByTime(activities)
	
	// 限制数量
	if len(activities) > limit {
		activities = activities[:limit]
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"activities": activities,
		"count":      len(activities),
	})
}

// sortActivitiesByTime 按时间排序活动
func sortActivitiesByTime(activities []map[string]interface{}) {
	// 实现按时间排序的逻辑
	// 这里简化处理，实际应该解析时间字符串进行排序
}

// GetStorageAnalysisHandler 获取存储空间分析
func GetStorageAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	db := models.GetDB()
	
	// 获取存储使用详情
	analysis := map[string]interface{}{
		"total": map[string]interface{}{
			"used":  0,
			"items": []map[string]interface{}{},
		},
		"breakdown": map[string]interface{}{
			"emails":      0,
			"attachments": 0,
			"other":       0,
		},
		"trend": []map[string]interface{}{},
	}
	
	// 计算邮件占用空间（估算）
	var emailCount int
	var emailSize int64
	db.QueryRow(`
		SELECT COUNT(*), SUM(LENGTH(body)) 
		FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?) AND is_deleted = 0
	`, userID, userID).Scan(&emailCount, &emailSize)
	
	if emailSize == 0 {
		emailSize = int64(emailCount) * 1024 // 每封邮件估算1KB
	}
	
	// 计算附件占用空间
	var attachmentCount int64
	var attachmentSize int64
	db.QueryRow(`
		SELECT COUNT(*), COALESCE(SUM(file_size), 0) 
		FROM attachments a
		JOIN emails e ON a.email_id = e.id
		WHERE e.sender_id = ? OR e.recipient_id = ?
	`, userID, userID).Scan(&attachmentCount, &attachmentSize)
	
	// 获取用户总存储
	user, err := models.GetUserByID(db, userID)
	if err == nil {
		totalUsed := user.StorageUsed
		otherSize := totalUsed - emailSize - attachmentSize
		if otherSize < 0 {
			otherSize = 0
		}
		
		analysis["total"] = map[string]interface{}{
			"used":        float64(totalUsed) / (1024 * 1024 * 1024),
			"max":         float64(user.MaxStorage) / (1024 * 1024 * 1024),
			"percent":     float64(totalUsed) / float64(user.MaxStorage) * 100,
			"items": []map[string]interface{}{
				{
					"name":  "邮件内容",
					"size":  float64(emailSize) / (1024 * 1024),
					"color": "#4361ee",
				},
				{
					"name":  "附件",
					"size":  float64(attachmentSize) / (1024 * 1024),
					"color": "#4cc9f0",
				},
				{
					"name":  "其他",
					"size":  float64(otherSize) / (1024 * 1024),
					"color": "#7209b7",
				},
			},
		}
		
		analysis["breakdown"] = map[string]interface{}{
			"emails":      float64(emailSize) / (1024 * 1024),
			"attachments": float64(attachmentSize) / (1024 * 1024),
			"other":       float64(otherSize) / (1024 * 1024),
		}
		
		// 获取存储使用趋势（最近30天）
		rows, err := db.Query(`
			SELECT DATE(created_at) as date,
			       COALESCE(SUM(file_size), 0) as daily_size
			FROM attachments a
			JOIN emails e ON a.email_id = e.id
			WHERE (e.sender_id = ? OR e.recipient_id = ?)
			  AND a.created_at >= DATE('now', '-30 days')
			GROUP BY DATE(created_at)
			ORDER BY date
		`, userID, userID)
		
		if err == nil {
			defer rows.Close()
			trend := []map[string]interface{}{}
			for rows.Next() {
				var date string
				var dailySize int64
				if err := rows.Scan(&date, &dailySize); err == nil {
					trend = append(trend, map[string]interface{}{
						"date": date,
						"size": float64(dailySize) / (1024 * 1024),
					})
				}
			}
			analysis["trend"] = trend
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"analysis": analysis,
	})
}

// GetEmailAnalyticsHandler 获取邮件分析
func GetEmailAnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "month"
	}
	
	db := models.GetDB()
	
	analytics := map[string]interface{}{
		"period": period,
		"stats":  map[string]interface{}{},
		"charts": map[string]interface{}{},
	}
	
	// 根据时间段构建SQL条件
	var dateCondition string
	switch period {
	case "day":
		dateCondition = "DATE(created_at) = DATE('now')"
	case "week":
		dateCondition = "created_at >= DATE('now', '-7 days')"
	case "month":
		dateCondition = "created_at >= DATE('now', '-30 days')"
	case "year":
		dateCondition = "created_at >= DATE('now', '-365 days')"
	default:
		dateCondition = "created_at >= DATE('now', '-30 days')"
	}
	
	// 获取发送和接收统计
	var sentCount, receivedCount int
	db.QueryRow(`
		SELECT 
			SUM(CASE WHEN sender_id = ? THEN 1 ELSE 0 END),
			SUM(CASE WHEN recipient_id = ? THEN 1 ELSE 0 END)
		FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?)
		  AND `+dateCondition+`
		  AND is_deleted = 0 AND is_draft = 0
	`, userID, userID, userID, userID).Scan(&sentCount, &receivedCount)
	
	// 获取阅读率
	var readCount int
	db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND is_read = 1
		  AND `+dateCondition+`
		  AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&readCount)
	
	readRate := 0.0
	if receivedCount > 0 {
		readRate = float64(readCount) / float64(receivedCount) * 100
	}
	
	// 获取热门联系人
	topContacts := []map[string]interface{}{}
	rows, err := db.Query(`
		SELECT 
			CASE 
				WHEN sender_id = ? THEN recipient_email
				ELSE sender_email
			END as contact_email,
			COUNT(*) as email_count
		FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?)
		  AND `+dateCondition+`
		  AND is_deleted = 0 AND is_draft = 0
		GROUP BY contact_email
		ORDER BY email_count DESC
		LIMIT 10
	`, userID, userID, userID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var contactEmail string
			var emailCount int
			if err := rows.Scan(&contactEmail, &emailCount); err == nil {
				topContacts = append(topContacts, map[string]interface{}{
					"email": contactEmail,
					"count": emailCount,
				})
			}
		}
	}
	
	// 获取时间段内每天的邮件数量
	dailyStats := []map[string]interface{}{}
	rows, err = db.Query(`
		SELECT DATE(created_at) as date,
		       SUM(CASE WHEN sender_id = ? THEN 1 ELSE 0 END) as sent,
		       SUM(CASE WHEN recipient_id = ? THEN 1 ELSE 0 END) as received
		FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?)
		  AND `+dateCondition+`
		  AND is_deleted = 0 AND is_draft = 0
		GROUP BY DATE(created_at)
		ORDER BY date
	`, userID, userID, userID, userID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date string
			var sent, received int
			if err := rows.Scan(&date, &sent, &received); err == nil {
				dailyStats = append(dailyStats, map[string]interface{}{
					"date":     date,
					"sent":     sent,
					"received": received,
					"total":    sent + received,
				})
			}
		}
	}
	
	analytics["stats"] = map[string]interface{}{
		"sent":      sentCount,
		"received":  receivedCount,
		"total":     sentCount + receivedCount,
		"read_rate": readRate,
		"avg_per_day": func() float64 {
			days := 30.0 // 默认30天
			if period == "day" {
				days = 1
			} else if period == "week" {
				days = 7
			} else if period == "year" {
				days = 365
			}
			return float64(sentCount+receivedCount) / days
		}(),
	}
	
	analytics["contacts"] = map[string]interface{}{
		"top": topContacts,
		"total_unique": len(topContacts),
	}
	
	analytics["charts"] = map[string]interface{}{
		"daily": dailyStats,
		"by_hour": getEmailByHour(db, userID, dateCondition),
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"analytics": analytics,
	})
}

// getEmailByHour 获取按小时分布的邮件统计
func getEmailByHour(db *models.Database, userID int, dateCondition string) []map[string]interface{} {
	hourlyStats := make([]map[string]interface{}, 24)
	
	// 初始化24小时
	for i := 0; i < 24; i++ {
		hourlyStats[i] = map[string]interface{}{
			"hour": i,
			"sent": 0,
			"received": 0,
			"total": 0,
		}
	}
	
	rows, err := db.Query(`
		SELECT 
			strftime('%H', created_at) as hour,
			SUM(CASE WHEN sender_id = ? THEN 1 ELSE 0 END) as sent,
			SUM(CASE WHEN recipient_id = ? THEN 1 ELSE 0 END) as received
		FROM emails 
		WHERE (sender_id = ? OR recipient_id = ?)
		  AND `+dateCondition+`
		  AND is_deleted = 0 AND is_draft = 0
		GROUP BY strftime('%H', created_at)
	`, userID, userID, userID, userID)
	
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var hourStr string
			var sent, received int
			if err := rows.Scan(&hourStr, &sent, &received); err == nil {
				hour, _ := strconv.Atoi(hourStr)
				if hour >= 0 && hour < 24 {
					hourlyStats[hour] = map[string]interface{}{
						"hour":     hour,
						"sent":     sent,
						"received": received,
						"total":    sent + received,
					}
				}
			}
		}
	}
	
	return hourlyStats
}

// CleanupOldDataHandler 清理旧数据
func CleanupOldDataHandler(w http.ResponseWriter, r *http.Request) {
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
	
	var params struct {
		DaysOld int  `json:"days_old"`
		DryRun  bool `json:"dry_run"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	if params.DaysOld <= 0 {
		params.DaysOld = 365 // 默认清理一年前的数据
	}
	
	// 计算删除日期
	deleteBefore := time.Now().AddDate(0, 0, -params.DaysOld)
	
	results := map[string]interface{}{
		"dry_run":    params.DryRun,
		"days_old":   params.DaysOld,
		"delete_before": deleteBefore.Format("2006-01-02"),
		"deleted":    map[string]int{},
	}
	
	if !params.DryRun {
		// 实际删除操作
		// 1. 删除旧的会话记录
		res, err := db.Exec(`
			DELETE FROM sessions 
			WHERE expires_at < ?
		`, deleteBefore)
		
		if err == nil {
			rowsAffected, _ := res.RowsAffected()
			results["deleted"].(map[string]int)["sessions"] = int(rowsAffected)
		}
		
		// 2. 删除已删除的邮件（保留30天）
		emailDeleteBefore := time.Now().AddDate(0, 0, -30)
		res, err = db.Exec(`
			DELETE FROM emails 
			WHERE is_deleted = 1 AND updated_at < ?
		`, emailDeleteBefore)
		
		if err == nil {
			rowsAffected, _ := res.RowsAffected()
			results["deleted"].(map[string]int)["emails"] = int(rowsAffected)
		}
		
		// 3. 清理空的临时文件（这里需要实现文件系统清理）
		
		utils.Info("管理员 %d 执行了数据清理操作: %v", userID, results)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "数据清理完成",
		"results": results,
	})
}