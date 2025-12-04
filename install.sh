#!/bin/bash

# SwiftPost 安装脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 函数：打印彩色消息
print_color() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 函数：打印分隔线
print_separator() {
    echo "=================================================="
}

# 函数：检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_color $RED "❌ $1 未安装"
        return 1
    fi
    return 0
}

# 函数：检查是否为root用户
check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_color $RED "❌ 此脚本需要root权限运行"
        exit 1
    fi
}

# 函数：安装依赖
install_dependencies() {
    print_color $BLUE "📦 安装系统依赖..."
    
    # 检测操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    else
        print_color $RED "❌ 无法检测操作系统"
        exit 1
    fi
    
    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y \
                curl wget git build-essential \
                python3 python3-pip python3-venv \
                sqlite3 libsqlite3-dev \
                nginx certbot \
                redis-server \
                postgresql postgresql-contrib
            ;;
        
        centos|rhel|fedora)
            yum update -y
            yum install -y \
                curl wget git gcc make \
                python3 python3-pip python3-virtualenv \
                sqlite sqlite-devel \
                nginx certbot \
                redis postgresql postgresql-server
            ;;
        
        alpine)
            apk update
            apk add \
                curl wget git build-base \
                python3 py3-pip python3-dev \
                sqlite sqlite-dev \
                nginx certbot \
                redis postgresql postgresql-client
            ;;
        
        *)
            print_color $YELLOW "⚠️  不支持的操作系统: $OS"
            print_color $YELLOW "请手动安装以下依赖:"
            print_color $YELLOW "  - Go 1.21+"
            print_color $YELLOW "  - Python 3.8+"
            print_color $YELLOW "  - SQLite3"
            print_color $YELLOW "  - Git"
            ;;
    esac
    
    print_color $GREEN "✅ 系统依赖安装完成"
}

# 函数：安装Go
install_go() {
    if check_command "go"; then
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        if [[ $GO_VERSION > 1.21 ]]; then
            print_color $GREEN "✅ Go $GO_VERSION 已安装"
            return 0
        fi
    fi
    
    print_color $BLUE "🔧 安装 Go..."
    
    GO_VERSION="1.21.4"
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        armv7l) ARCH="armv6l" ;;
        *) print_color $RED "❌ 不支持的架构: $ARCH"; exit 1 ;;
    esac
    
    # 下载Go
    GO_TAR="go${GO_VERSION}.linux-${ARCH}.tar.gz"
    wget -q "https://golang.org/dl/${GO_TAR}" -O /tmp/$GO_TAR
    
    # 解压
    tar -C /usr/local -xzf /tmp/$GO_TAR
    rm /tmp/$GO_TAR
    
    # 设置环境变量
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export GOPATH=$HOME/go' >> /etc/profile
    echo 'export PATH=$PATH:$GOPATH/bin' >> /etc/profile
    
    source /etc/profile
    
    print_color $GREEN "✅ Go $GO_VERSION 安装完成"
}

# 函数：创建系统用户
create_user() {
    if id "swiftpost" &>/dev/null; then
        print_color $BLUE "👤 SwiftPost 用户已存在"
        return 0
    fi
    
    print_color $BLUE "👤 创建 SwiftPost 系统用户..."
    
    useradd -r -s /bin/false -m -d /opt/swiftpost swiftpost
    usermod -aG swiftpost www-data
    
    print_color $GREEN "✅ 用户创建完成"
}

# 函数：安装SwiftPost
install_swiftpost() {
    print_color $BLUE "🚀 安装 SwiftPost..."
    
    # 创建目录
    mkdir -p /opt/swiftpost/{data,logs,ssl,backups}
    chown -R swiftpost:swiftpost /opt/swiftpost
    chmod 755 /opt/swiftpost
    
    # 克隆代码
    if [ ! -d "/opt/swiftpost/.git" ]; then
        git clone https://github.com/byusiteam/swiftpost.git /opt/swiftpost/app
        chown -R swiftpost:swiftpost /opt/swiftpost/app
    fi
    
    cd /opt/swiftpost/app
    
    # 安装Go依赖
    print_color $BLUE "📦 安装 Go 依赖..."
    cd backend/go
    go mod download
    go build -o /opt/swiftpost/swiftpost
    cd ../..
    
    # 安装Python依赖
    print_color $BLUE "🐍 安装 Python 依赖..."
    pip3 install -r backend/python/requirements.txt
    
    # 创建配置文件
    if [ ! -f "/opt/swiftpost/config.json" ]; then
        print_color $BLUE "📝 创建配置文件..."
        cp config.example.json /opt/swiftpost/config.json
        
        # 生成随机的JWT密钥
        JWT_SECRET=$(openssl rand -base64 48)
        sed -i "s/\"your-secret-key-change-this-in-production\"/\"$JWT_SECRET\"/" /opt/swiftpost/config.json
        
        # 更新路径
        sed -i "s|\"data/swiftpost.db\"|\"/opt/swiftpost/data/swiftpost.db\"|" /opt/swiftpost/config.json
        sed -i "s|\"data/emails\"|\"/opt/swiftpost/data/emails\"|" /opt/swiftpost/config.json
        sed -i "s|\"data/attachments\"|\"/opt/swiftpost/data/attachments\"|" /opt/swiftpost/config.json
    fi
    
    # 创建数据库
    print_color $BLUE "🗄️  初始化数据库..."
    sudo -u swiftpost python3 start.py --init-only
    
    print_color $GREEN "✅ SwiftPost 安装完成"
}

# 函数：配置SSL证书
configure_ssl() {
    print_color $BLUE "🔐 配置 SSL 证书..."
    
    read -p "请输入域名 (例如: mail.example.com): " DOMAIN
    
    if [ -z "$DOMAIN" ]; then
        print_color $YELLOW "⚠️  未提供域名，使用自签名证书"
        
        # 生成自签名证书
        openssl req -x509 -newkey rsa:4096 \
            -keyout /opt/swiftpost/ssl/key.pem \
            -out /opt/swiftpost/ssl/cert.pem \
            -days 365 -nodes -subj "/CN=swiftpost.local"
        
        # 更新配置文件
        sed -i 's/"enabled": false/"enabled": true/' /opt/swiftpost/config.json
        sed -i 's|"cert": ""|"cert": "/opt/swiftpost/ssl/cert.pem"|' /opt/swiftpost/config.json
        sed -i 's|"key": ""|"key": "/opt/swiftpost/ssl/key.pem"|' /opt/swiftpost/config.json
        
    else
        print_color $BLUE "📝 获取 Let's Encrypt 证书..."
        
        # 使用certbot获取证书
        certbot certonly --standalone \
            -d $DOMAIN \
            --non-interactive \
            --agree-tos \
            --email admin@$DOMAIN
        
        if [ $? -eq 0 ]; then
            # 证书路径
            CERT_PATH="/etc/letsencrypt/live/$DOMAIN/fullchain.pem"
            KEY_PATH="/etc/letsencrypt/live/$DOMAIN/privkey.pem"
            
            # 创建符号链接
            ln -sf $CERT_PATH /opt/swiftpost/ssl/cert.pem
            ln -sf $KEY_PATH /opt/swiftpost/ssl/key.pem
            
            # 更新配置文件
            sed -i "s/\"swiftpost.local\"/\"$DOMAIN\"/" /opt/swiftpost/config.json
            sed -i 's/"enabled": false/"enabled": true/' /opt/swiftpost/config.json
            sed -i "s|\"cert\": \"\"|\"cert\": \"$CERT_PATH\"|" /opt/swiftpost/config.json
            sed -i "s|\"key\": \"\"|\"key\": \"$KEY_PATH\"|" /opt/swiftpost/config.json
            
            # 设置证书自动续期
            echo "0 0 * * * certbot renew --quiet --post-hook \"systemctl reload nginx\"" >> /etc/crontab
            
            print_color $GREEN "✅ SSL 证书配置完成"
        else
            print_color $YELLOW "⚠️  证书获取失败，使用自签名证书"
            configure_ssl_self_signed
        fi
    fi
    
    chown -R swiftpost:swiftpost /opt/swiftpost/ssl
    chmod 600 /opt/swiftpost/ssl/*.pem
}

# 函数：配置系统服务
configure_service() {
    print_color $BLUE "⚙️  配置系统服务..."
    
    # 复制服务文件
    cp /opt/swiftpost/app/systemd/swiftpost.service /etc/systemd/system/
    cp /opt/swiftpost/app/systemd/swiftpost.env /etc/swiftpost/
    
    # 更新环境文件
    sed -i "s|/opt/swiftpost|/opt/swiftpost|g" /etc/swiftpost/swiftpost.env
    
    # 重新加载systemd
    systemctl daemon-reload
    systemctl enable swiftpost
    
    # 配置Nginx
    if [ -f "/opt/swiftpost/app/nginx/nginx.conf" ]; then
        cp /opt/swiftpost/app/nginx/nginx.conf /etc/nginx/
        cp -r /opt/swiftpost/app/nginx/conf.d/* /etc/nginx/conf.d/
        
        # 更新域名配置
        DOMAIN=$(grep '"domain"' /opt/swiftpost/config.json | awk -F'"' '{print $4}')
        sed -i "s/swiftpost.local/$DOMAIN/g" /etc/nginx/conf.d/swiftpost.conf
        
        systemctl enable nginx
    fi
    
    print_color $GREEN "✅ 系统服务配置完成"
}

# 函数：启动服务
start_services() {
    print_color $BLUE "🚀 启动服务..."
    
    systemctl start swiftpost
    systemctl start nginx
    
    # 检查服务状态
    if systemctl is-active --quiet swiftpost; then
        print_color $GREEN "✅ SwiftPost 服务启动成功"
    else
        print_color $RED "❌ SwiftPost 服务启动失败"
        journalctl -u swiftpost -n 50 --no-pager
    fi
    
    if systemctl is-active --quiet nginx; then
        print_color $GREEN "✅ Nginx 服务启动成功"
    else
        print_color $RED "❌ Nginx 服务启动失败"
        journalctl -u nginx -n 50 --no-pager
    fi
}

# 函数：显示安装信息
show_installation_info() {
    print_separator
    print_color $GREEN "🎉 SwiftPost 安装完成！"
    print_separator
    
    # 获取配置信息
    DOMAIN=$(grep '"domain"' /opt/swiftpost/config.json | awk -F'"' '{print $4}')
    PORT=$(grep '"port"' /opt/swiftpost/config.json | awk -F'"' '{print $4}')
    SSL_ENABLED=$(grep '"enabled"' /opt/swiftpost/config.json | head -1 | awk -F': ' '{print $2}' | tr -d ',')
    
    print_color $CYAN "📋 安装信息:"
    print_color $CYAN "  - 安装目录: /opt/swiftpost"
    print_color $CYAN "  - 数据目录: /opt/swiftpost/data"
    print_color $CYAN "  - 日志目录: /opt/swiftpost/logs"
    print_color $CYAN "  - 配置文件: /opt/swiftpost/config.json"
    
    print_color $MAGENTA "🌐 访问信息:"
    if [ "$SSL_ENABLED" = "true" ]; then
        print_color $MAGENTA "  - 主地址: https://$DOMAIN"
        print_color $MAGENTA "  - 备用地址: https://$DOMAIN:$PORT"
    else
        print_color $MAGENTA "  - 主地址: http://$DOMAIN:$PORT"
    fi
    
    print_color $YELLOW "🔧 管理命令:"
    print_color $YELLOW "  - 启动服务: systemctl start swiftpost"
    print_color $YELLOW "  - 停止服务: systemctl stop swiftpost"
    print_color $YELLOW "  - 重启服务: systemctl restart swiftpost"
    print_color $YELLOW "  - 查看日志: journalctl -u swiftpost -f"
    
    print_color $BLUE "📖 后续步骤:"
    print_color $BLUE "  1. 访问网站完成初始设置"
    print_color $BLUE "  2. 第一个注册的用户将成为管理员"
    print_color $BLUE "  3. 配置DNS记录指向服务器"
    print_color $BLUE "  4. 设置防火墙规则"
    
    print_separator
    print_color $GREEN "💡 提示: 更多信息请查看 /opt/swiftpost/README.md"
    print_separator
}

# 主函数
main() {
    print_separator
    print_color $CYAN "🚀 SwiftPost 安装脚本"
    print_color $BLUE "📅 版本: 1.0.0"
    print_color $MAGENTA "🏢 组织: ByUsi Team"
    print_separator
    
    # 检查root权限
    check_root
    
    # 安装步骤
    install_dependencies
    install_go
    create_user
    install_swiftpost
    configure_ssl
    configure_service
    start_services
    show_installation_info
    
    print_color $GREEN "✅ 安装完成！"
}

# 运行主函数
main "$@"