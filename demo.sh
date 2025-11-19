#!/bin/bash

set -e

echo "🚀 SmartCI Bash任务调度演示"
echo "================================"

# 检查SmartCI是否编译
if [ ! -f "./smart-ci" ]; then
    echo "❌ SmartCI未编译，正在编译..."
    go build -o smart-ci .
fi

# 创建演示日志目录
mkdir -p ./logs

echo "✅ SmartCI已准备就绪"
echo ""
echo "📋 当前配置的Bash任务："
echo "1. backup-database - 数据库备份 (每天凌晨2点)"
echo "2. cleanup-logs - 日志清理 (每周日午夜)"
echo "3. system-monitor - 系统监控 (每5分钟)"
echo "4. deploy-app - 应用部署 (周一、三、五早上6点)"
echo "5. sync-data - 数据同步 (每4小时)"
echo ""

# 演示手动触发bash任务
echo "🔧 演示手动触发Bash任务..."

echo "1. 触发日志清理任务..."
curl -s "http://localhost:8080/webhook/bash?task=cleanup-logs" 2>/dev/null || {
    echo "❌ SmartCI服务未启动，请先运行: ./smart-ci"
    exit 1
}

echo "✅ 任务已触发，查看日志..."
sleep 2

# 查看最新的日志文件
LATEST_LOG=$(ls -t ./logs/bash-cleanup-logs-*.log 2>/dev/null | head -1)
if [ -n "$LATEST_LOG" ]; then
    echo "📄 最新日志内容："
    echo "--------------------------------"
    cat "$LATEST_LOG"
    echo "--------------------------------"
else
    echo "❌ 未找到日志文件"
fi

echo ""
echo "🌐 API端点："
echo "- Webhook触发: curl http://localhost:8080/webhook/bash?task=<任务名>"
echo "- MCP工具列表: curl http://localhost:8080/mcp/tools"
echo "- 健康检查: curl http://localhost:8080/webhook"
echo ""

echo "📖 更多信息请查看: docs/bash-tasks.md"