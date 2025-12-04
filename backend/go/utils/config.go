package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Server struct {
		Host   string `json:"host"`
		Port   string `json:"port"`
		Domain string `json:"domain"`
		SSL    struct {
			Enabled bool   `json:"enabled"`
			Cert    string `json:"cert"`
			Key     string `json:"key"`
		} `json:"ssl"`
	} `json:"server"`
	
	Database struct {
		Path          string `json:"path"`
		PythonEnabled bool   `json:"python_enabled"`
		PythonScript  string `json:"python_script"`
	} `json:"database"`
	
	Email struct {
		StoragePath    string `json:"storage_path"`
		MaxEmailSize   int64  `json:"max_email_size"`
		DefaultDomain  string `json:"default_domain"`
		AttachmentPath string `json:"attachment_path"`
	} `json:"email"`
	
	Security struct {
		JWTSecret      string `json:"jwt_secret"`
		TokenExpiry    int    `json:"token_expiry"`
		RateLimit      int    `json:"rate_limit"`
		CorsOrigins    string `json:"cors_origins"`
	} `json:"security"`
	
	Admin struct {
		FirstUserAdmin bool `json:"first_user_admin"`
	} `json:"admin"`
	
	WebSocket struct {
		Enabled        bool `json:"enabled"`
		PingInterval   int  `json:"ping_interval"`
		MaxMessageSize int  `json:"max_message_size"`
	} `json:"websocket"`
}

func LoadConfig(filename string) (*Config, error) {
	config := &Config{}
	
	// 检查配置文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// 创建默认配置
		config = createDefaultConfig()
		
		// 保存默认配置到文件
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("无法序列化默认配置: %v", err)
		}
		
		if err := os.WriteFile(filename, data, 0644); err != nil {
			return nil, fmt.Errorf("无法创建配置文件: %v", err)
		}
		
		PrintColored("📝 已创建默认配置文件: "+filename, 0, ColorGreen)
		return config, nil
	}
	
	// 读取配置文件
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}
	
	// 解析 JSON
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("无法解析配置文件: %v", err)
	}
	
	return config, nil
}

func createDefaultConfig() *Config {
	config := &Config{}
	
	// 服务器配置
	config.Server.Host = "0.0.0.0"
	config.Server.Port = "252"
	config.Server.Domain = "swiftpost.local"
	config.Server.SSL.Enabled = false
	config.Server.SSL.Cert = ""
	config.Server.SSL.Key = ""
	
	// 数据库配置
	config.Database.Path = "data/swiftpost.db"
	config.Database.PythonEnabled = true
	config.Database.PythonScript = "start.py"
	
	// 邮件配置
	config.Email.StoragePath = "data/emails"
	config.Email.MaxEmailSize = 26214400 // 25MB
	config.Email.DefaultDomain = "{username}:{id}.swiftpost.local"
	config.Email.AttachmentPath = "data/attachments"
	
	// 安全配置
	config.Security.JWTSecret = "your-secret-key-change-this-in-production"
	config.Security.TokenExpiry = 72 // 小时
	config.Security.RateLimit = 100
	config.Security.CorsOrigins = "*"
	
	// 管理员配置
	config.Admin.FirstUserAdmin = true
	
	// WebSocket 配置
	config.WebSocket.Enabled = true
	config.WebSocket.PingInterval = 30
	config.WebSocket.MaxMessageSize = 1024 * 1024 // 1MB
	
	return config
}

func (c *Config) Save(filename string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filename, data, 0644)
}