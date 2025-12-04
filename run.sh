#!/bin/bash

# SwiftPost 启动脚本

set -e

echo "=================================================="
echo "🚀 启动 SwiftPost 邮件服务"
echo "=================================================="

# 检查依赖
echo "🔍 检查系统依赖..."
command -v go >/dev/null 2>&1 || { echo "❌ Go 未安装"; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "❌ Python3 未安装"; exit 1; }

echo "✅ 依赖检查通过"

# 创建必要的目录
echo "📁 创建数据目录..."
mkdir -p data/emails
mkdir -p data/attachments

# 检查配置文件
if [ ! -f "config.json" ]; then
    echo "📝 创建默认配置文件..."
    cat > config.json << EOF
{
  "server": {
    "host": "0.0.0.0",
    "port": "252",
    "domain": "swiftpost.local",
    "ssl": {
      "enabled": false,
      "cert": "",
      "key": ""
    }
  },
  "database": {
    "path": "data/swiftpost.db",
    "python_enabled": true,
    "python_script": "start.py"
  },
  "email": {
    "storage_path": "data/emails",
    "max_email_size": 26214400,
    "default_domain": "{username}:{id}.swiftpost.local",
    "attachment_path": "data/attachments"
  },
  "security": {
    "jwt_secret": "your-secret-key-change-this-in-production",
    "token_expiry": 72,
    "rate_limit": 100,
    "cors_origins": "*"
  },
  "admin": {
    "first_user_admin": true
  },
  "websocket": {
    "enabled": true,
    "ping_interval": 30,
    "max_message_size": 1048576
  }
}
EOF
    echo "✅ 配置文件已创建"
fi

# 安装Go依赖
echo "📦 安装Go依赖..."
cd backend/go
go mod download
cd ../..

# 启动Python数据库服务
echo "🐍 启动Python数据库服务..."
python3 start.py --child &
PYTHON_PID=$!

# 等待Python服务启动
sleep 2

# 启动Go服务
echo "🚀 启动Go主服务..."
cd backend/go
go run main.go

# 清理
echo "🔄 停止服务..."
kill $PYTHON_PID 2>/dev/null || true
wait $PYTHON_PID 2>/dev/null || true

echo "👋 SwiftPost 服务已停止"