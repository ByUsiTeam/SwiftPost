FROM golang:1.21-alpine AS builder

WORKDIR /app

# 安装依赖
RUN apk add --no-cache python3 py3-pip sqlite

# 复制Go模块文件
COPY backend/go/go.mod backend/go/go.sum ./
RUN go mod download

# 复制源代码
COPY backend/go/ ./
COPY frontend/ ../frontend/
COPY config.json ../
COPY start.py ../
COPY backend/python/ ../backend/python/

# 创建必要的目录
RUN mkdir -p /app/data/emails /app/data/attachments

# 安装Python依赖
RUN pip3 install --no-cache-dir sqlite3

# 构建Go应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o swiftpost .

# 最终镜像
FROM alpine:latest

RUN apk --no-cache add ca-certificates python3 sqlite

WORKDIR /app

# 从构建阶段复制文件
COPY --from=builder /app/swiftpost .
COPY --from=builder /app/../frontend ./frontend
COPY --from=builder /app/../config.json .
COPY --from=builder /app/../start.py .
COPY --from=builder /app/../backend/python ./backend/python

# 创建数据目录
RUN mkdir -p /app/data/emails /app/data/attachments

# 设置权限
RUN chmod +x swiftpost start.py

# 暴露端口
EXPOSE 252

# 启动脚本
CMD ["/bin/sh", "-c", "python3 start.py --child & sleep 2 && ./swiftpost"]