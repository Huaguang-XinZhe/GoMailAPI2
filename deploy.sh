#!/bin/bash

# GoMailAPI2 服务器端部署脚本
# 用于在服务器上更新和重启服务

set -e # 遇到错误立即退出

echo "=========================================="
echo "GoMailAPI2 自动部署脚本"
echo "时间: $(date)"
echo "=========================================="

# 项目目录
PROJECT_DIR="/home/ubuntu/app/GoMailAPI2"
COMPOSE_FILE="compose.prod.yaml"

# 检查目录是否存在
if [ ! -d "$PROJECT_DIR" ]; then
    echo "❌ 错误: 项目目录 $PROJECT_DIR 不存在"
    exit 1
fi

cd "$PROJECT_DIR"

echo "📁 当前目录: $(pwd)"

# 拉取最新代码
echo "🔄 拉取最新代码..."
git pull origin main

# 检查 Docker Compose 文件是否存在
if [ ! -f "$COMPOSE_FILE" ]; then
    echo "❌ 错误: $COMPOSE_FILE 文件不存在"
    exit 1
fi

# 停止当前服务
echo "🛑 停止当前服务..."
docker compose -f "$COMPOSE_FILE" down

# 清理旧镜像（释放空间）
echo "🧹 清理旧镜像..."
docker image prune -a -f

# 拉取最新镜像
echo "📥 拉取最新镜像..."
docker compose -f "$COMPOSE_FILE" pull

# 启动服务
echo "🚀 启动服务..."
docker compose -f "$COMPOSE_FILE" up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 10

# 检查服务状态
echo "📊 服务状态:"
docker compose -f "$COMPOSE_FILE" ps

# 检查日志
echo "📋 最近日志:"
docker compose -f "$COMPOSE_FILE" logs --tail=20

echo "=========================================="
echo "✅ 部署完成！"
echo "时间: $(date)"
echo "=========================================="
