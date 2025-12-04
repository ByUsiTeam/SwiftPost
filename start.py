#!/usr/bin/env python3
"""
SwiftPost 启动脚本
当单独启动时自动结束进程
当作为子进程启动时持续运行
"""

import sys
import os
import time
import sqlite3
from pathlib import Path

def init_database():
    """初始化数据库"""
    db_path = "data/swiftpost.db"
    
    # 确保目录存在
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    
    conn = sqlite3.connect(db_path)
    cursor = conn.cursor()
    
    # 创建用户表
    cursor.execute('''
    CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        username TEXT UNIQUE NOT NULL,
        email TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        is_admin BOOLEAN DEFAULT 0,
        custom_domain TEXT,
        storage_used INTEGER DEFAULT 0,
        max_storage INTEGER DEFAULT 1073741824,
        is_active BOOLEAN DEFAULT 1,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
    ''')
    
    # 创建邮件表
    cursor.execute('''
    CREATE TABLE IF NOT EXISTS emails (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        uuid TEXT UNIQUE NOT NULL,
        sender_id INTEGER NOT NULL,
        recipient_id INTEGER NOT NULL,
        sender_email TEXT NOT NULL,
        recipient_email TEXT NOT NULL,
        subject TEXT NOT NULL,
        body TEXT NOT NULL,
        is_read BOOLEAN DEFAULT 0,
        is_starred BOOLEAN DEFAULT 0,
        is_deleted BOOLEAN DEFAULT 0,
        is_draft BOOLEAN DEFAULT 0,
        has_attachment BOOLEAN DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (sender_id) REFERENCES users (id),
        FOREIGN KEY (recipient_id) REFERENCES users (id)
    )
    ''')
    
    # 创建附件表
    cursor.execute('''
    CREATE TABLE IF NOT EXISTS attachments (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        email_id INTEGER NOT NULL,
        uuid TEXT UNIQUE NOT NULL,
        filename TEXT NOT NULL,
        filepath TEXT NOT NULL,
        file_size INTEGER,
        mime_type TEXT,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (email_id) REFERENCES emails (id)
    )
    ''')
    
    # 创建会话表
    cursor.execute('''
    CREATE TABLE IF NOT EXISTS sessions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        user_id INTEGER NOT NULL,
        session_token TEXT UNIQUE NOT NULL,
        ip_address TEXT,
        user_agent TEXT,
        expires_at TIMESTAMP NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        FOREIGN KEY (user_id) REFERENCES users (id)
    )
    ''')
    
    # 创建索引
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_emails_recipient ON emails(recipient_id, created_at DESC)')
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_emails_sender ON emails(sender_id, created_at DESC)')
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)')
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)')
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_emails_uuid ON emails(uuid)')
    cursor.execute('CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(session_token)')
    
    # 检查是否有管理员用户
    cursor.execute('SELECT COUNT(*) FROM users WHERE is_admin = 1')
    admin_count = cursor.fetchone()[0]
    
    if admin_count == 0:
        print("⚠️  没有管理员用户，第一个注册的用户将成为管理员")
    
    conn.commit()
    conn.close()
    
    return db_path

def monitor_database():
    """监控数据库连接"""
    db_path = "data/swiftpost.db"
    
    while True:
        try:
            conn = sqlite3.connect(db_path)
            cursor = conn.cursor()
            cursor.execute('SELECT 1')
            conn.close()
            time.sleep(10)  # 每10秒检查一次
        except KeyboardInterrupt:
            print("\n🔄 数据库监控已停止")
            break
        except Exception as e:
            print(f"❌ 数据库连接错误: {e}")
            time.sleep(5)

if __name__ == "__main__":
    print("=" * 50)
    print("🚀 SwiftPost 数据库服务")
    print("=" * 50)
    
    # 检查是否作为子进程启动
    if len(sys.argv) > 1 and sys.argv[1] == "--child":
        print("📊 数据库服务已启动（子进程模式）")
        print("📁 数据库文件: data/swiftpost.db")
        
        # 初始化数据库
        db_path = init_database()
        print(f"✅ 数据库初始化完成: {db_path}")
        
        # 持续监控
        monitor_database()
    else:
        print("❌ 此脚本只能作为子进程启动")
        print("💡 请使用 Go 主程序启动此服务")
        sys.exit(1)