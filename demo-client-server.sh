#!/bin/bash

# SmartCI Client-Server 架构演示脚本

echo "🚀 SmartCI Client-Server 架构演示"
echo "=================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 函数：打印带颜色的消息
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查构建文件是否存在
if [ ! -f "./smart-ci-server" ] || [ ! -f "./smart-ci-client" ]; then
    print_info "构建SmartCI组件..."
    make build
fi

# 启动服务器
print_info "启动SmartCI服务器..."
./smart-ci-server -mode server -config config.yaml &
SERVER_PID=$!

# 等待服务器启动
sleep 3

# 检查服务器是否启动成功
if ! curl -s http://localhost:8080/health > /dev/null; then
    print_error "服务器启动失败"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

print_success "服务器启动成功 (PID: $SERVER_PID)"

echo ""
echo "🔧 演示Client-Server通信"
echo "========================="

# 1. 健康检查
print_info "1. 检查服务器健康状态..."
./smart-ci-client -command "health"

echo ""

# 2. 列出所有任务
print_info "2. 列出所有可用任务..."
./smart-ci-client -command "list"

echo ""

# 3. 查看配置
print_info "3. 查看服务器配置..."
./smart-ci-client -command "config"

echo ""

# 4. 运行一个简单的任务
print_info "4. 运行cleanup-logs任务..."
./smart-ci-client -command "run cleanup-logs"

echo ""

# 5. 检查任务状态
print_info "5. 检查任务状态..."
./smart-ci-client -command "status"

echo ""

# 6. 查看任务日志
print_info "6. 查看任务日志 (最近10行)..."
./smart-ci-client -command "logs cleanup-logs 10"

echo ""

# 7. 测试API直接调用
print_info "7. 直接调用HTTP API..."
curl -s -X POST http://localhost:8080/api/command \
    -H "Content-Type: application/json" \
    -d '{"command": "health"}' | jq .

echo ""

# 8. 测试兼容性接口
print_info "8. 测试兼容性的MCP接口..."
curl -s http://localhost:8080/mcp/tools | jq '.[0].name'

echo ""

# 停止服务器
print_info "停止服务器..."
./smart-ci-client -command "server-down"

# 等待服务器停止
sleep 2

# 强制清理进程
kill $SERVER_PID 2>/dev/null

print_success "演示完成！"

echo ""
echo "📋 功能特性总结"
echo "==============="
echo "✅ Client-Server架构分离"
echo "✅ 远程命令执行"
echo "✅ 丰富的命令集 (run, start, stop, status, logs, config, health, list)"
echo "✅ HTTP API接口"
echo "✅ 配置文件服务器配置"
echo "✅ 向后兼容原有接口"
echo "✅ 优雅的启动和停止"
echo "✅ 认证支持 (可选)"
echo "✅ Makefile构建支持"

echo ""
echo "🎯 使用示例"
echo "=========="
echo "# 启动服务器"
echo "./smart-ci-server -mode server -config config.yaml"
echo ""
echo "# 使用客户端"
echo "./smart-ci-client -command 'list'"
echo "./smart-ci-client -command 'run backup-database'"
echo "./smart-ci-client -command 'status'"
echo ""
echo "# 直接API调用"
echo "curl -X POST http://localhost:8080/api/command \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"command\": \"health\"}'"