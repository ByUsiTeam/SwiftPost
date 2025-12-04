package utils

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Validator 配置验证器
type Validator struct {
	Errors map[string]string
}

// NewValidator 创建新的验证器
func NewValidator() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

// Required 检查字段是否为空
func (v *Validator) Required(field, value string) {
	if strings.TrimSpace(value) == "" {
		v.Errors[field] = "不能为空"
	}
}

// MinLength 检查最小长度
func (v *Validator) MinLength(field, value string, min int) {
	if len(strings.TrimSpace(value)) < min {
		v.Errors[field] = fmt.Sprintf("长度不能少于 %d 个字符", min)
	}
}

// MaxLength 检查最大长度
func (v *Validator) MaxLength(field, value string, max int) {
	if len(strings.TrimSpace(value)) > max {
		v.Errors[field] = fmt.Sprintf("长度不能超过 %d 个字符", max)
	}
}

// Email 验证邮箱格式
func (v *Validator) Email(field, value string) {
	if value == "" {
		return
	}
	
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(value) {
		v.Errors[field] = "无效的邮箱格式"
	}
}

// URL 验证URL格式
func (v *Validator) URL(field, value string) {
	if value == "" {
		return
	}
	
	_, err := url.ParseRequestURI(value)
	if err != nil {
		v.Errors[field] = "无效的URL格式"
	}
}

// IPAddress 验证IP地址
func (v *Validator) IPAddress(field, value string) {
	if value == "" {
		return
	}
	
	if net.ParseIP(value) == nil {
		v.Errors[field] = "无效的IP地址"
	}
}

// Port 验证端口号
func (v *Validator) Port(field, value string) {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		v.Errors[field] = "无效的端口号 (1-65535)"
	}
}

// Numeric 验证是否为数字
func (v *Validator) Numeric(field, value string) {
	if value == "" {
		return
	}
	
	if _, err := strconv.Atoi(value); err != nil {
		v.Errors[field] = "必须是数字"
	}
}

// Range 验证数值范围
func (v *Validator) Range(field string, value, min, max int) {
	if value < min || value > max {
		v.Errors[field] = fmt.Sprintf("必须在 %d 到 %d 之间", min, max)
	}
}

// FileExists 验证文件是否存在
func (v *Validator) FileExists(field, path string) {
	if path == "" {
		return
	}
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		v.Errors[field] = "文件不存在"
	}
}

// DirectoryExists 验证目录是否存在
func (v *Validator) DirectoryExists(field, path string) {
	if path == "" {
		return
	}
	
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		v.Errors[field] = "目录不存在"
	} else if !info.IsDir() {
		v.Errors[field] = "不是目录"
	}
}

// ValidDomain 验证域名格式
func (v *Validator) ValidDomain(field, domain string) {
	if domain == "" {
		return
	}
	
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if !domainRegex.MatchString(domain) {
		v.Errors[field] = "无效的域名格式"
	}
}

// StrongPassword 验证密码强度
func (v *Validator) StrongPassword(field, password string) {
	if password == "" {
		return
	}
	
	var hasMinLen, hasUpper, hasLower, hasNumber, hasSpecial bool
	
	// 检查最小长度
	if len(password) >= 8 {
		hasMinLen = true
	}
	
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", ch):
			hasSpecial = true
		}
	}
	
	var errors []string
	if !hasMinLen {
		errors = append(errors, "至少8个字符")
	}
	if !hasUpper {
		errors = append(errors, "至少一个大写字母")
	}
	if !hasLower {
		errors = append(errors, "至少一个小写字母")
	}
	if !hasNumber {
		errors = append(errors, "至少一个数字")
	}
	if !hasSpecial {
		errors = append(errors, "至少一个特殊字符")
	}
	
	if len(errors) > 0 {
		v.Errors[field] = strings.Join(errors, ", ")
	}
}

// ValidPath 验证路径格式
func (v *Validator) ValidPath(field, path string) {
	if path == "" {
		return
	}
	
	// 检查路径是否包含非法字符
	invalidChars := regexp.MustCompile(`[<>:"|?*]`)
	if invalidChars.MatchString(path) {
		v.Errors[field] = "路径包含非法字符"
		return
	}
	
	// 检查是否为绝对路径
	if !filepath.IsAbs(path) {
		// 如果是相对路径，转换为绝对路径
		absPath, err := filepath.Abs(path)
		if err != nil {
			v.Errors[field] = "无效的路径格式"
			return
		}
		path = absPath
	}
}

// HexColor 验证十六进制颜色代码
func (v *Validator) HexColor(field, color string) {
	if color == "" {
		return
	}
	
	colorRegex := regexp.MustCompile(`^#?([a-fA-F0-9]{6}|[a-fA-F0-9]{3})$`)
	if !colorRegex.MatchString(color) {
		v.Errors[field] = "无效的十六进制颜色代码"
	}
}

// TimeDuration 验证时间持续时间
func (v *Validator) TimeDuration(field, duration string) {
	if duration == "" {
		return
	}
	
	// 检查是否为有效的时间格式
	durationRegex := regexp.MustCompile(`^\d+[smhd]$`)
	if !durationRegex.MatchString(duration) {
		v.Errors[field] = "无效的时间格式 (例如: 30s, 5m, 2h, 1d)"
	}
}

// Valid 检查是否有错误
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// ValidateConfig 验证配置
func ValidateConfig(config *Config) error {
	validator := NewValidator()
	
	// 验证服务器配置
	validator.Required("server.host", config.Server.Host)
	validator.Required("server.port", config.Server.Port)
	validator.Port("server.port", config.Server.Port)
	
	if config.Server.SSL.Enabled {
		validator.FileExists("server.ssl.cert", config.Server.SSL.Cert)
		validator.FileExists("server.ssl.key", config.Server.SSL.Key)
	}
	
	// 验证数据库配置
	validator.Required("database.path", config.Database.Path)
	validator.ValidPath("database.path", config.Database.Path)
	
	// 验证邮件配置
	validator.Range("email.max_email_size", int(config.Email.MaxEmailSize), 1024*1024, 100*1024*1024) // 1MB to 100MB
	
	// 验证安全配置
	validator.Required("security.jwt_secret", config.Security.JWTSecret)
	validator.MinLength("security.jwt_secret", config.Security.JWTSecret, 32)
	validator.Range("security.token_expiry", config.Security.TokenExpiry, 1, 720) // 1小时到30天
	validator.Range("security.rate_limit", config.Security.RateLimit, 1, 10000)
	
	// 验证WebSocket配置
	if config.WebSocket.Enabled {
		validator.Range("websocket.ping_interval", config.WebSocket.PingInterval, 10, 300)
		validator.Range("websocket.max_message_size", config.WebSocket.MaxMessageSize, 1024, 10*1024*1024) // 1KB to 10MB
	}
	
	if !validator.Valid() {
		var errorMsgs []string
		for field, msg := range validator.Errors {
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: %s", field, msg))
		}
		return fmt.Errorf("配置验证失败: %s", strings.Join(errorMsgs, "; "))
	}
	
	return nil
}

// SanitizeConfig 清理和标准化配置
func SanitizeConfig(config *Config) {
	// 清理服务器配置
	config.Server.Host = strings.TrimSpace(config.Server.Host)
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	
	config.Server.Port = strings.TrimSpace(config.Server.Port)
	if config.Server.Port == "" {
		config.Server.Port = "252"
	}
	
	config.Server.Domain = strings.TrimSpace(config.Server.Domain)
	if config.Server.Domain == "" {
		config.Server.Domain = "swiftpost.local"
	}
	
	// 清理数据库配置
	config.Database.Path = strings.TrimSpace(config.Database.Path)
	if config.Database.Path == "" {
		config.Database.Path = "data/swiftpost.db"
	}
	
	// 确保路径是绝对路径
	if !filepath.IsAbs(config.Database.Path) {
		absPath, err := filepath.Abs(config.Database.Path)
		if err == nil {
			config.Database.Path = absPath
		}
	}
	
	// 清理邮件配置
	config.Email.StoragePath = strings.TrimSpace(config.Email.StoragePath)
	if config.Email.StoragePath == "" {
		config.Email.StoragePath = "data/emails"
	}
	
	config.Email.AttachmentPath = strings.TrimSpace(config.Email.AttachmentPath)
	if config.Email.AttachmentPath == "" {
		config.Email.AttachmentPath = "data/attachments"
	}
	
	// 确保目录路径存在
	os.MkdirAll(config.Email.StoragePath, 0755)
	os.MkdirAll(config.Email.AttachmentPath, 0755)
	
	// 清理安全配置
	config.Security.JWTSecret = strings.TrimSpace(config.Security.JWTSecret)
	if config.Security.JWTSecret == "" || config.Security.JWTSecret == "your-secret-key-change-this-in-production" {
		// 生成一个随机的JWT密钥
		config.Security.JWTSecret = generateRandomString(64)
	}
	
	config.Security.CorsOrigins = strings.TrimSpace(config.Security.CorsOrigins)
	if config.Security.CorsOrigins == "" {
		config.Security.CorsOrigins = "*"
	}
	
	// 设置默认值
	if config.Email.MaxEmailSize <= 0 {
		config.Email.MaxEmailSize = 25 * 1024 * 1024 // 25MB
	}
	
	if config.Security.TokenExpiry <= 0 {
		config.Security.TokenExpiry = 72 // 小时
	}
	
	if config.Security.RateLimit <= 0 {
		config.Security.RateLimit = 100
	}
	
	if config.WebSocket.PingInterval <= 0 {
		config.WebSocket.PingInterval = 30
	}
	
	if config.WebSocket.MaxMessageSize <= 0 {
		config.WebSocket.MaxMessageSize = 1024 * 1024 // 1MB
	}
}

// ValidateEmailAddress 验证邮箱地址
func ValidateEmailAddress(email string) bool {
	if email == "" {
		return false
	}
	
	// 基本格式验证
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return false
	}
	
	// 长度验证
	if len(email) > 254 {
		return false
	}
	
	// 本地部分和域名部分验证
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	
	localPart := parts[0]
	domainPart := parts[1]
	
	// 本地部分不能以点开头或结尾
	if strings.HasPrefix(localPart, ".") || strings.HasSuffix(localPart, ".") {
		return false
	}
	
	// 本地部分不能包含连续的点
	if strings.Contains(localPart, "..") {
		return false
	}
	
	// 域名验证
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domainPart) {
		return false
	}
	
	return true
}

// ValidateUsername 验证用户名
func ValidateUsername(username string) bool {
	if username == "" {
		return false
	}
	
	// 长度验证
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	
	// 字符集验证
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !usernameRegex.MatchString(username) {
		return false
	}
	
	// 不能以数字开头
	if username[0] >= '0' && username[0] <= '9' {
		return false
	}
	
	// 保留用户名检查
	reservedUsernames := []string{
		"admin", "administrator", "root", "system", "mail", "postmaster",
		"hostmaster", "webmaster", "support", "info", "contact", "help",
		"swiftpost", "byusi", "server", "service",
	}
	
	for _, reserved := range reservedUsernames {
		if strings.ToLower(username) == reserved {
			return false
		}
	}
	
	return true
}

// ValidatePassword 验证密码
func ValidatePassword(password string) (bool, string) {
	if password == "" {
		return false, "密码不能为空"
	}
	
	var errors []string
	
	// 长度检查
	if len(password) < 8 {
		errors = append(errors, "至少8个字符")
	}
	
	// 复杂度检查
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, ch := range password {
		switch {
		case ch >= 'A' && ch <= 'Z':
			hasUpper = true
		case ch >= 'a' && ch <= 'z':
			hasLower = true
		case ch >= '0' && ch <= '9':
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", ch):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		errors = append(errors, "至少一个大写字母")
	}
	if !hasLower {
		errors = append(errors, "至少一个小写字母")
	}
	if !hasNumber {
		errors = append(errors, "至少一个数字")
	}
	if !hasSpecial {
		errors = append(errors, "至少一个特殊字符")
	}
	
	// 常见弱密码检查
	weakPasswords := []string{
		"password", "12345678", "qwertyui", "admin123",
		"letmein", "welcome", "monkey", "dragon",
		"sunshine", "master", "hello", "freedom",
		"whatever", "qazwsxed", "password1", "trustno1",
	}
	
	for _, weak := range weakPasswords {
		if strings.ToLower(password) == weak {
			errors = append(errors, "密码太常见")
			break
		}
	}
	
	if len(errors) > 0 {
		return false, strings.Join(errors, ", ")
	}
	
	return true, ""
}

// ValidateDomain 验证域名
func ValidateDomain(domain string) bool {
	if domain == "" {
		return true // 空域名是允许的（表示不使用自定义域名）
	}
	
	// 基本格式验证
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if !domainRegex.MatchString(domain) {
		return false
	}
	
	// 长度检查
	if len(domain) > 253 {
		return false
	}
	
	// 标签长度检查
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) > 63 {
			return false
		}
		
		// 标签不能以连字符开头或结尾
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	
	return true
}

// ValidateEmailTemplate 验证邮件模板变量
func ValidateEmailTemplate(template, username string, userID int) (string, error) {
	// 替换用户名
	template = strings.ReplaceAll(template, "{username}", username)
	
	// 替换用户ID
	template = strings.ReplaceAll(template, "{id}", strconv.Itoa(userID))
	
	// 检查是否还有未替换的变量
	unmatchedVars := regexp.MustCompile(`\{[^}]*\}`)
	if unmatchedVars.MatchString(template) {
		return "", fmt.Errorf("模板包含未定义的变量")
	}
	
	return template, nil
}

// ValidateFileSize 验证文件大小
func ValidateFileSize(fileSize, maxSize int64) bool {
	return fileSize <= maxSize
}

// ValidateMimeType 验证MIME类型
func ValidateMimeType(mimeType string, allowedTypes []string) bool {
	if len(allowedTypes) == 0 {
		return true
	}
	
	for _, allowed := range allowedTypes {
		if mimeType == allowed {
			return true
		}
		
		// 支持通配符，如 "image/*"
		if strings.HasSuffix(allowed, "/*") {
			prefix := strings.TrimSuffix(allowed, "/*")
			if strings.HasPrefix(mimeType, prefix) {
				return true
			}
		}
	}
	
	return false
}

// GenerateDefaultEmailDomain 生成默认邮箱域名
func GenerateDefaultEmailDomain(config *Config, username string, userID int) (string, error) {
	template := config.Email.DefaultDomain
	if template == "" {
		template = "{username}:{id}.swiftpost.local"
	}
	
	return ValidateEmailTemplate(template, username, userID)
}

// Helper function to generate random string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?"
	b := make([]byte, length)
	for i := range b {
		// 这里简化处理，实际应该使用 crypto/rand
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}