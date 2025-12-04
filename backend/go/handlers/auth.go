package handlers

import (
	"SwiftPost/models"
	"SwiftPost/utils"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
	User    *UserResponse `json:"user,omitempty"`
}

type UserResponse struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	IsAdmin      bool   `json:"is_admin"`
	CustomDomain string `json:"custom_domain"`
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error("注册请求解析失败: %v", err)
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "无效的请求格式",
		})
		return
	}
	
	// 验证输入
	if strings.TrimSpace(req.Username) == "" {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "用户名不能为空",
		})
		return
	}
	
	if strings.TrimSpace(req.Email) == "" {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "邮箱不能为空",
		})
		return
	}
	
	if strings.TrimSpace(req.Password) == "" {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "密码不能为空",
		})
		return
	}
	
	if len(req.Password) < 6 {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "密码长度至少6位",
		})
		return
	}
	
	// 检查邮箱是否已注册
	db := models.GetDB()
	existingUser, err := models.GetUserByEmail(db, req.Email)
	if err == nil && existingUser != nil {
		respondJSON(w, http.StatusConflict, AuthResponse{
			Success: false,
			Message: "邮箱已被注册",
		})
		return
	}
	
	// 检查用户名是否已存在
	existingUser, err = models.GetUserByUsername(db, req.Username)
	if err == nil && existingUser != nil {
		respondJSON(w, http.StatusConflict, AuthResponse{
			Success: false,
			Message: "用户名已被使用",
		})
		return
	}
	
	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.Error("密码哈希失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	// 创建用户
	userID, err := models.CreateUser(db, req.Username, req.Email, string(hashedPassword))
	if err != nil {
		utils.Error("创建用户失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "创建用户失败",
		})
		return
	}
	
	utils.Info("新用户注册: %s (%s)", req.Username, req.Email)
	
	// 生成 JWT token
	config, _ := utils.LoadConfig("config.json")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"username": req.Username,
		"email": req.Email,
		"exp": time.Now().Add(time.Hour * time.Duration(config.Security.TokenExpiry)).Unix(),
		"iat": time.Now().Unix(),
	})
	
	tokenString, err := token.SignedString([]byte(config.Security.JWTSecret))
	if err != nil {
		utils.Error("生成Token失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	// 获取用户信息
	user, err := models.GetUserByID(db, int(userID))
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, AuthResponse{
		Success: true,
		Token:   tokenString,
		Message: "注册成功",
		User: &UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
			CustomDomain: user.CustomDomain,
		},
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error("登录请求解析失败: %v", err)
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "无效的请求格式",
		})
		return
	}
	
	// 验证输入
	if strings.TrimSpace(req.Email) == "" {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "邮箱不能为空",
		})
		return
	}
	
	if strings.TrimSpace(req.Password) == "" {
		respondJSON(w, http.StatusBadRequest, AuthResponse{
			Success: false,
			Message: "密码不能为空",
		})
		return
	}
	
	// 查找用户
	db := models.GetDB()
	user, err := models.GetUserByEmail(db, req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			respondJSON(w, http.StatusUnauthorized, AuthResponse{
				Success: false,
				Message: "邮箱或密码错误",
			})
			return
		}
		utils.Error("查询用户失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	// 检查用户是否激活
	if !user.IsActive {
		respondJSON(w, http.StatusForbidden, AuthResponse{
			Success: false,
			Message: "账号已被禁用",
		})
		return
	}
	
	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "邮箱或密码错误",
		})
		return
	}
	
	utils.Info("用户登录: %s (%s)", user.Username, user.Email)
	
	// 生成 JWT token
	config, _ := utils.LoadConfig("config.json")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"is_admin": user.IsAdmin,
		"exp":      time.Now().Add(time.Hour * time.Duration(config.Security.TokenExpiry)).Unix(),
		"iat":      time.Now().Unix(),
	})
	
	tokenString, err := token.SignedString([]byte(config.Security.JWTSecret))
	if err != nil {
		utils.Error("生成Token失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, AuthResponse{
		Success: true,
		Token:   tokenString,
		Message: "登录成功",
		User: &UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			IsAdmin:  user.IsAdmin,
			CustomDomain: user.CustomDomain,
		},
	})
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	// 在实际应用中，这里应该将Token加入黑名单
	// 由于JWT是无状态的，客户端需要自行删除Token
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "登出成功",
	})
}

func RefreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "缺少授权Token",
		})
		return
	}
	
	// 提取Token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "无效的Token格式",
		})
		return
	}
	
	// 解析并验证Token
	config, _ := utils.LoadConfig("config.json")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.NewValidationError("无效的签名方法", jwt.ValidationErrorSignatureInvalid)
		}
		return []byte(config.Security.JWTSecret), nil
	})
	
	if err != nil {
		utils.Error("Token解析失败: %v", err)
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "无效的Token",
		})
		return
	}
	
	if !token.Valid {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "Token已失效",
		})
		return
	}
	
	// 获取用户信息
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		respondJSON(w, http.StatusUnauthorized, AuthResponse{
			Success: false,
			Message: "无效的Token声明",
		})
		return
	}
	
	// 生成新的Token
	newToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  claims["user_id"],
		"username": claims["username"],
		"email":    claims["email"],
		"is_admin": claims["is_admin"],
		"exp":      time.Now().Add(time.Hour * time.Duration(config.Security.TokenExpiry)).Unix(),
		"iat":      time.Now().Unix(),
	})
	
	newTokenString, err := newToken.SignedString([]byte(config.Security.JWTSecret))
	if err != nil {
		utils.Error("生成新Token失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, AuthResponse{
			Success: false,
			Message: "服务器内部错误",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, AuthResponse{
		Success: true,
		Token:   newTokenString,
		Message: "Token刷新成功",
	})
}

func GetProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取用户信息失败",
		})
		return
	}
	
	// 统计未读邮件数量
	var unreadCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM emails 
		WHERE recipient_id = ? AND is_read = 0 AND is_deleted = 0 AND is_draft = 0
	`, userID).Scan(&unreadCount)
	if err != nil {
		unreadCount = 0
	}
	
	// 计算存储使用情况
	storageUsedMB := float64(user.StorageUsed) / (1024 * 1024)
	maxStorageMB := float64(user.MaxStorage) / (1024 * 1024)
	storagePercent := 0.0
	if maxStorageMB > 0 {
		storagePercent = (storageUsedMB / maxStorageMB) * 100
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"user": map[string]interface{}{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"is_admin":      user.IsAdmin,
			"custom_domain": user.CustomDomain,
			"storage": map[string]interface{}{
				"used":       storageUsedMB,
				"max":        maxStorageMB,
				"percent":    storagePercent,
				"used_bytes": user.StorageUsed,
				"max_bytes":  user.MaxStorage,
			},
			"is_active":   user.IsActive,
			"created_at":  user.CreatedAt,
			"stats": map[string]interface{}{
				"unread_emails": unreadCount,
			},
		},
	})
}

func UpdateProfileHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	var updateData struct {
		Username     string `json:"username"`
		CustomDomain string `json:"custom_domain"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	db := models.GetDB()
	user, err := models.GetUserByID(db, userID)
	if err != nil {
		utils.Error("获取用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "获取用户信息失败",
		})
		return
	}
	
	// 更新用户名（如果提供了新用户名）
	if updateData.Username != "" && updateData.Username != user.Username {
		// 检查用户名是否已存在
		existingUser, err := models.GetUserByUsername(db, updateData.Username)
		if err == nil && existingUser != nil && existingUser.ID != userID {
			respondJSON(w, http.StatusConflict, map[string]interface{}{
				"success": false,
				"message": "用户名已被使用",
			})
			return
		}
		user.Username = updateData.Username
	}
	
	// 更新自定义域名
	user.CustomDomain = updateData.CustomDomain
	
	// 保存更改
	if err := models.UpdateUser(db, user); err != nil {
		utils.Error("更新用户信息失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "更新用户信息失败",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "个人信息更新成功",
		"user": map[string]interface{}{
			"id":            user.ID,
			"username":      user.Username,
			"email":         user.Email,
			"is_admin":      user.IsAdmin,
			"custom_domain": user.CustomDomain,
		},
	})
}

func UpdateDomainHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	
	var req struct {
		Domain string `json:"domain"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "无效的请求格式",
		})
		return
	}
	
	// 验证域名格式（简单的验证）
	if req.Domain != "" {
		if !strings.Contains(req.Domain, ".") {
			respondJSON(w, http.StatusBadRequest, map[string]interface{}{
				"success": false,
				"message": "无效的域名格式",
			})
			return
		}
	}
	
	db := models.GetDB()
	if err := models.UpdateCustomDomain(db, userID, req.Domain); err != nil {
		utils.Error("更新域名失败: %v", err)
		respondJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "更新域名失败",
		})
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "域名更新成功",
		"domain":  req.Domain,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}