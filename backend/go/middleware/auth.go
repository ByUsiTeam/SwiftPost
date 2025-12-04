package middleware

import (
	"SwiftPost/utils"
	"context"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
)

// AuthMiddleware 验证JWT令牌
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// 尝试从Cookie获取
			cookie, err := r.Cookie("token")
			if err != nil {
				// 重定向到登录页面
				http.Redirect(w, r, "/login", http.StatusFound)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		config, _ := utils.LoadConfig("config.json")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("无效的签名方法", jwt.ValidationErrorSignatureInvalid)
			}
			return []byte(config.Security.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			utils.Error("Token验证失败: %v", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// 提取用户信息
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		userID := int(userIDFloat)

		username, _ := claims["username"].(string)
		email, _ := claims["email"].(string)
		isAdmin, _ := claims["is_admin"].(bool)

		// 将用户信息添加到上下文
		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "username", username)
		ctx = context.WithValue(ctx, "email", email)
		ctx = context.WithValue(ctx, "is_admin", isAdmin)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AdminMiddleware 检查是否是管理员
func AdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAdmin, ok := r.Context().Value("is_admin").(bool)
		if !ok || !isAdmin {
			http.Error(w, "需要管理员权限", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// APIAuthMiddleware API认证中间件
func APIAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.Error("API请求缺少认证头")
			respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"message": "需要认证",
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			utils.Error("API请求Token格式错误")
			respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"message": "无效的Token格式",
			})
			return
		}

		config, _ := utils.LoadConfig("config.json")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.NewValidationError("无效的签名方法", jwt.ValidationErrorSignatureInvalid)
			}
			return []byte(config.Security.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			utils.Error("API Token验证失败: %v", err)
			respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"message": "无效的Token",
			})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"message": "无效的Token声明",
			})
			return
		}

		// 提取用户信息
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			respondJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"success": false,
				"message": "无效的用户ID",
			})
			return
		}
		userID := int(userIDFloat)

		username, _ := claims["username"].(string)
		email, _ := claims["email"].(string)
		isAdmin, _ := claims["is_admin"].(bool)

		// 将用户信息添加到上下文
		ctx := context.WithValue(r.Context(), "user_id", userID)
		ctx = context.WithValue(ctx, "username", username)
		ctx = context.WithValue(ctx, "email", email)
		ctx = context.WithValue(ctx, "is_admin", isAdmin)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// APIAdminMiddleware API管理员中间件
func APIAdminMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		isAdmin, ok := r.Context().Value("is_admin").(bool)
		if !ok || !isAdmin {
			respondJSON(w, http.StatusForbidden, map[string]interface{}{
				"success": false,
				"message": "需要管理员权限",
			})
			return
		}
		next.ServeHTTP(w, r)
	}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// 需要导入 encoding/json
import "encoding/json"