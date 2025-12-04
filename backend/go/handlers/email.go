package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
)

type EmailRequest struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type EmailResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	EmailID int    `json:"email_id,omitempty"`
}

func SendEmailHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 解析 multipart/form-data 请求
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		utils.Error("解析表单数据失败: %v", err)
		respondJSON(w, http.StatusBadRequest, EmailResponse{
			Success: false,
			Message: "请求数据太大或格式错误",
		})
		return
	}
	
	// 获取表单数据
	to := r.FormValue("to")
	subject := r.FormValue("subject")
	body := r.FormValue("body")
	
	if to == "" || subject == "" || body == "" {
		respondJSON(w, http.StatusBadRequest, EmailResponse{
			Success: false,
			Message: "收件人、主题和内容不能为空",
		})
		return
	}
	
	// 获取发件人信息
	db := models.GetDB()
	sender, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取发件人信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, EmailResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	// 查找收件人
	recipient, err := models.GetUserByEmail(db, to)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusBadRequest, EmailResponse{
				Success: false,
				Message: "收件人邮箱不存在",
			})
			return
		}
		utils.Error("查找收件人失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, EmailResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	// 创建邮件
	email := &models.Email{
		UUID:           uuid.New().String(),
		SenderID:       sender.ID,
		RecipientID:    recipient.ID,
		SenderEmail:    sender.Email,
		RecipientEmail: recipient.Email,
		Subject:        subject,
		Body:           body,
		IsRead:         false,
		IsStarred:      false,
		IsDeleted:      false,
		IsDraft:        false,
		HasAttachment:  false,
	}
	
	// 处理附件
	hasAttachment := false
	file, handler, err := r.FormFile("attachment")
	if err == nil {
		defer file.Close()
		
		// 检查文件大小
		config, _ := utils.LoadConfig("config.json")
		if handler.Size > config.Email.MaxEmailSize {
			respondJSON(w, http.StatusBadRequest, EmailResponse{
				Success: false,
				Message: "附件太大",
			})
			return
		}
		
		// 创建附件目录
		attachmentDir := config.Email.AttachmentPath
		if err := os.MkdirAll(attachmentDir, 0755); err != nil {
			utils.Error("创建附件目录失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, EmailResponse{
				Success: false,
				Message: "服务器内部错误",
			})
			return
		}
		
		// 生成唯一文件名
		fileExt := filepath.Ext(handler.Filename)
		fileName := uuid.New().String() + fileExt
		filePath := filepath.Join(attachmentDir, fileName)
		
		// 保存文件
		dst, err := os.Create(filePath)
		if err != nil {
			utils.Error("创建文件失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, EmailResponse{
				Success: false,
				Message: "服务器内部错误",
			})
			return
		}
		defer dst.Close()
		
		if _, err := io.Copy(dst, file); err != nil {
			utils.Error("保存文件失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, EmailResponse{
				Success: false,
				Message: "服务器内部错误",
			})
			return
		}
		
		hasAttachment = true
		email.HasAttachment = true
	}
	
	// 保存邮件到数据库
	emailID, err := models.CreateEmail(db, email)
	if err != nil {
		utils.Error("保存邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, EmailResponse{
			Success: false,
			Message: "发送邮件失败",
		})
		return
	}
	
	// 如果有附件，保存附件信息
	if hasAttachment && file != nil {
		attachment := &models.Attachment{
			EmailID:  int(emailID),
			UUID:     uuid.New().String(),
			Filename: handler.Filename,
			Filepath: filePath,
			FileSize: handler.Size,
			MimeType: handler.Header.Get("Content-Type"),
		}
		
		if _, err := models.CreateAttachment(db, attachment); err != nil {
			utils.Error("保存附件信息失败: %v", err)
			// 继续执行，不返回错误
		}
		
		// 更新用户存储使用量
		sender.StorageUsed += handler.Size
		if err := models.UpdateUserStorage(db, sender.ID, sender.StorageUsed); err != nil {
			utils.Error("更新存储使用量失败: %v", err)
		}
	}
	
	utils.Info("邮件发送: %s -> %s (主题: %s)", sender.Email, recipient.Email, subject)
	
	// 通过WebSocket通知收件人
	go notifyNewEmail(recipient.ID, emailID)
	
	respondJSON(w, http.StatusOK, EmailResponse{
		Success: true,
		Message: "邮件发送成功",
		EmailID: int(emailID),
	})
}

func GetEmailsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	// 获取查询参数
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "inbox"
	}
	
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	
	offset := (page - 1) * limit
	
	db := models.GetDB()
	
	// 获取邮件列表
	emails, err := models.GetEmailsByRecipient(db, userID, limit, offset, folder)
	if err != nil {
		utils.Error("获取邮件列表失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取邮件列表失败",
		})
		return
	}
	
	// 获取总数
	total, err := models.CountEmailsByRecipient(db, userID, folder)
	if err != nil {
		utils.Error("统计邮件数量失败: %v", err)
		total = len(emails)
	}
	
	// 获取发件人姓名
	emailList := make([]map[string]interface{}, len(emails))
	for i, email := range emails {
		sender, err := models.GetUserByID(db, email.SenderID)
		senderName := email.SenderEmail
		if err == nil {
			senderName = sender.Username
		}
		
		recipient, err := models.GetUserByID(db, email.RecipientID)
		recipientName := email.RecipientEmail
		if err == nil {
			recipientName = recipient.Username
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
			"body_preview":    getBodyPreview(email.Body),
			"is_read":         email.IsRead,
			"is_starred":      email.IsStarred,
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
		"folder": folder,
	})
}

func GetEmailHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	emailID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的邮件ID",
		})
		return
	}
	
	db := models.GetDB()
	
	// 获取邮件
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "邮件不存在",
			})
			return
		}
		utils.Error("获取邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取邮件失败",
		})
		return
	}
	
	// 检查权限：用户必须是收件人或发件人
	if email.RecipientID != userID && email.SenderID != userID {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "无权访问此邮件",
		})
		return
	}
	
	// 如果是收件人且未读，标记为已读
	if email.RecipientID == userID && !email.IsRead {
		if err := models.MarkAsRead(db, email.ID); err != nil {
			utils.Error("标记邮件已读失败: %v", err)
		} else {
			email.IsRead = true
		}
	}
	
	// 获取发件人和收件人信息
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
	
	// 获取附件
	attachments, _ := models.GetAttachmentsByEmail(db, email.ID)
	attachmentList := make([]map[string]interface{}, len(attachments))
	for i, att := range attachments {
		attachmentList[i] = map[string]interface{}{
			"id":        att.ID,
			"uuid":      att.UUID,
			"filename":  att.Filename,
			"file_size": att.FileSize,
			"mime_type": att.MimeType,
			"created_at": att.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
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
			"body":            email.Body,
			"is_read":         email.IsRead,
			"is_starred":      email.IsStarred,
			"is_draft":        email.IsDraft,
			"has_attachment":  email.HasAttachment,
			"created_at":      email.CreatedAt.Format("2006-01-02 15:04:05"),
			"time_ago":        getTimeAgo(email.CreatedAt),
			"attachments":     attachmentList,
		},
	})
}

func UpdateEmailHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	emailID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的邮件ID",
		})
		return
	}
	
	var updateData struct {
		IsDraft    *bool `json:"is_draft"`
		IsStarred  *bool `json:"is_starred"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	db := models.GetDB()
	
	// 获取邮件
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "邮件不存在",
			})
			return
		}
		utils.Error("获取邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取邮件失败",
		})
		return
	}
	
	// 检查权限：用户必须是发件人才能更新草稿
	if email.SenderID != userID {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "无权修改此邮件",
		})
		return
	}
	
	// 更新邮件
	updated := false
	if updateData.IsDraft != nil {
		email.IsDraft = *updateData.IsDraft
		updated = true
	}
	if updateData.IsStarred != nil {
		email.IsStarred = *updateData.IsStarred
		updated = true
	}
	
	if updated {
		// 这里应该有一个 UpdateEmail 函数，简化处理
		_, err = db.Exec(`
			UPDATE emails SET 
				is_draft = ?, 
				is_starred = ?, 
				updated_at = ?
			WHERE id = ?
		`, email.IsDraft, email.IsStarred, time.Now(), email.ID)
		
		if err != nil {
			utils.Error("更新邮件失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "更新邮件失败",
			})
			return
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "邮件更新成功",
	})
}

func DeleteEmailHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	emailID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的邮件ID",
		})
		return
	}
	
	permanent := r.URL.Query().Get("permanent") == "true"
	
	db := models.GetDB()
	
	// 获取邮件
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "邮件不存在",
			})
			return
		}
		utils.Error("获取邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取邮件失败",
		})
		return
	}
	
	// 检查权限：用户必须是收件人或发件人
	if email.RecipientID != userID && email.SenderID != userID {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "无权删除此邮件",
		})
		return
	}
	
	if permanent {
		// 永久删除
		if err := models.DeletePermanently(db, email.ID); err != nil {
			utils.Error("永久删除邮件失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "删除邮件失败",
			})
			return
		}
	} else {
		// 移动到回收站
		if err := models.MoveToTrash(db, email.ID); err != nil {
			utils.Error("移动邮件到回收站失败: %v", err)
			respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
				"success": false,
				"message": "删除邮件失败",
			})
			return
		}
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "邮件删除成功",
	})
}

func MarkAsReadHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	emailID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的邮件ID",
		})
		return
	}
	
	db := models.GetDB()
	
	// 获取邮件
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "邮件不存在",
			})
			return
		}
		utils.Error("获取邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取邮件失败",
		})
		return
	}
	
	// 检查权限：用户必须是收件人
	if email.RecipientID != userID {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "无权标记此邮件",
		})
		return
	}
	
	// 标记为已读
	if err := models.MarkAsRead(db, email.ID); err != nil {
		utils.Error("标记邮件已读失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "标记邮件失败",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "邮件已标记为已读",
	})
}

func ToggleStarHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	emailID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的邮件ID",
		})
		return
	}
	
	db := models.GetDB()
	
	// 获取邮件
	email, err := models.GetEmailByID(db, emailID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "邮件不存在",
			})
			return
		}
		utils.Error("获取邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取邮件失败",
		})
		return
	}
	
	// 检查权限：用户必须是收件人或发件人
	if email.RecipientID != userID && email.SenderID != userID {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "无权标记此邮件",
		})
		return
	}
	
	// 切换星标状态
	if err := models.ToggleStar(db, email.ID); err != nil {
		utils.Error("切换星标状态失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "标记邮件失败",
		})
		return
	}
	
	// 获取更新后的状态
	updatedEmail, _ := models.GetEmailByID(db, emailID)
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "星标状态已更新",
		"is_starred": updatedEmail.IsStarred,
	})
}

func UploadAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		utils.Error("解析表单数据失败: %v", err)
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "请求数据太大或格式错误",
		})
		return
	}
	
	// 获取文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		utils.Error("获取文件失败: %v", err)
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "没有上传文件",
		})
		return
	}
	defer file.Close()
	
	// 检查文件大小
	config, _ := utils.LoadConfig("config.json")
	if handler.Size > config.Email.MaxEmailSize {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "文件太大",
		})
		return
	}
	
	// 检查用户存储空间
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	if user.StorageUsed+handler.Size > user.MaxStorage {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "存储空间不足",
		})
		return
	}
	
	// 创建附件目录
	attachmentDir := config.Email.AttachmentPath
	if err := os.MkdirAll(attachmentDir, 0755); err != nil {
		utils.Error("创建附件目录失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	// 生成唯一文件名
	fileExt := filepath.Ext(handler.Filename)
	fileName := uuid.New().String() + fileExt
	filePath := filepath.Join(attachmentDir, fileName)
	
	// 保存文件
	dst, err := os.Create(filePath)
	if err != nil {
		utils.Error("创建文件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	defer dst.Close()
	
	if _, err := io.Copy(dst, file); err != nil {
		utils.Error("保存文件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	// 更新用户存储使用量
	user.StorageUsed += handler.Size
	if err := models.UpdateUserStorage(db, user.ID, user.StorageUsed); err != nil {
		utils.Error("更新存储使用量失败: %v", err)
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "文件上传成功",
		"file": map[string]interface{}{
			"uuid":      fileName,
			"filename":  handler.Filename,
			"file_size": handler.Size,
			"mime_type": handler.Header.Get("Content-Type"),
			"url":       fmt.Sprintf("/api/attachments/%s/download", fileName),
		},
	})
}

func DownloadAttachmentHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	vars := mux.Vars(r)
	attachmentUUID := vars["id"]
	
	db := models.GetDB()
	
	// 获取附件信息
	attachment, err := models.GetAttachmentByUUID(db, attachmentUUID)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusNotFound, map[string]interface{}{
				"success": false,
				"message": "附件不存在",
			})
			return
		}
		utils.Error("获取附件信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	// 获取邮件
	email, err := models.GetEmailByID(db, attachment.EmailID)
	if err != nil {
		utils.Error("获取邮件失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "服务器内部错误",
		})
		return
	}
	
	// 检查权限：用户必须是收件人或发件人
	if email.RecipientID != userID && email.SenderID != userID {
		respondJSON(w, http.StatusForbidden, map[string]interface{}{
			"success": false,
			"message": "无权下载此附件",
		})
		return
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(attachment.Filepath); os.IsNotExist(err) {
		respondJSON(w, http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "文件不存在",
		})
		return
	}
	
	// 设置下载头
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
	w.Header().Set("Content-Type", attachment.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(attachment.FileSize, 10))
	
	// 提供文件下载
	http.ServeFile(w, r, attachment.Filepath)
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

func notifyNewEmail(recipientID, emailID int) {
	// 这里应该通过WebSocket通知用户有新邮件
	// 简化实现，实际应用中应该发送WebSocket消息
	utils.Debug("新邮件通知: 收件人ID=%d, 邮件ID=%d", recipientID, emailID)
}