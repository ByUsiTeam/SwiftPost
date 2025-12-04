#!/usr/bin/env python3
"""
Python 数据库服务启动器
"""

import subprocess
import sys
import os

def main():
    """主函数"""
    # 检查数据库文件是否存在
    db_path = "data/swiftpost.db"
    
    if not os.path.exists(db_path):
        print("🔄 初始化数据库...")
        # 运行数据库初始化
        from start import init_database
        init_database()
    
    print("🐍 Python 数据库服务准备就绪")
    print("📊 使用 Ctrl+C 停止服务")

if __name__ == "__main__":
    main()