package models

import (
	"strconv"
	"github.com/gorilla/mux"
)

// GetIDFromRequest 从请求中获取ID参数
func GetIDFromRequest(r *http.Request, paramName string) (int, error) {
	vars := mux.Vars(r)
	idStr := vars[paramName]
	return strconv.Atoi(idStr)
}

// Pagination 分页结构
type Pagination struct {
	Page      int `json:"page"`
	Limit     int `json:"limit"`
	Total     int `json:"total"`
	TotalPage int `json:"total_page"`
}

// NewPagination 创建分页
func NewPagination(page, limit, total int) *Pagination {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	
	totalPage := (total + limit - 1) / limit
	if totalPage < 1 {
		totalPage = 1
	}
	
	return &Pagination{
		Page:      page,
		Limit:     limit,
		Total:     total,
		TotalPage: totalPage,
	}
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	// 简单的邮箱验证
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	
	at := strings.Index(email, "@")
	if at == -1 || at == 0 || at == len(email)-1 {
		return false
	}
	
	dot := strings.LastIndex(email[at:], ".")
	if dot == -1 || dot < 2 {
		return false
	}
	
	return true
}

// ValidateUsername 验证用户名格式
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	
	// 只允许字母、数字、下划线
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || 
			(c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	
	return true
}

// ValidatePassword 验证密码格式
func ValidatePassword(password string) bool {
	if len(password) < 6 {
		return false
	}
	
	// 检查是否包含至少一个字母和一个数字
	hasLetter := false
	hasDigit := false
	
	for _, c := range password {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	
	return hasLetter && hasDigit
}

// 需要导入的包
import (
	"net/http"
	"strings"
)