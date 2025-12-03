#!/bin/bash

echo "=========================================="
echo "测试 Metrics 数据收集功能"
echo "=========================================="
echo ""

# 构建项目
echo "📦 构建项目..."
go build -o smart-ci-server main.go
go build -o smart-ci-metrics ./cmd/metrics/main.go

if [ $? -ne 0 ]; then
    echo "❌ 构建失败"
    exit 1
fi
echo "✅ 构建成功"
echo ""

# 创建测试配置文件
echo "📝 创建测试配置..."
cat > test-config.yaml << 'EOF'
server:
  host: localhost
  port: 8081

bash_tasks:
  - name: test-metrics-success
    description: "测试成功任务"
    command: |
      echo "开始执行任务..."
      sleep 2
      echo "任务执行完成"
      exit 0
    timeout: 30

  - name: test-metrics-failure
    description: "测试失败任务"
    command: |
      echo "开始执行任务..."
      sleep 1
      echo "任务执行失败"
      exit 1
    timeout: 30

schedule: "@every 1h"
EOF

echo "✅ 配置文件创建成功"
echo ""

# 启动服务器
echo "🚀 启动服务器..."
./smart-ci-server -config test-config.yaml > /tmp/smart-ci-test.log 2>&1 &
SERVER_PID=$!

# 等待服务器启动
sleep 3

# 检查服务器是否运行
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ 服务器启动失败"
    cat /tmp/smart-ci-test.log
    exit 1
fi

echo "✅ 服务器已启动 (PID: $SERVER_PID)"
echo ""

# 执行测试任务
echo "🔧 执行测试任务..."
echo ""

echo "1️⃣ 执行成功任务..."
curl -s "http://localhost:8081/webhook/bash?task=test-metrics-success" > /dev/null
sleep 3

echo "2️⃣ 执行失败任务..."
curl -s "http://localhost:8081/webhook/bash?task=test-metrics-failure" > /dev/null
sleep 3

echo "3️⃣ 再次执行成功任务..."
curl -s "http://localhost:8081/webhook/bash?task=test-metrics-success" > /dev/null
sleep 3

echo ""
echo "✅ 任务执行完成"
echo ""

# 关闭服务器
echo "🛑 关闭服务器..."
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null
echo "✅ 服务器已关闭"
echo ""

# 检查元数据文件
echo "=========================================="
echo "📊 检查生成的元数据"
echo "=========================================="
echo ""

METADATA_COUNT=$(find ./logs -name "metadata.json" | wc -l)
echo "找到 $METADATA_COUNT 个元数据文件"
echo ""

if [ "$METADATA_COUNT" -gt 0 ]; then
    echo "=========================================="
    echo "📈 查看任务统计"
    echo "=========================================="
    echo ""
    
    # 显示所有任务
    echo "【所有任务概览】"
    ./smart-ci-metrics all
    echo ""
    
    # 显示成功任务详情
    echo "【成功任务详情】"
    ./smart-ci-metrics latest -task test-metrics-success
    echo ""
    
    ./smart-ci-metrics stats -task test-metrics-success
    echo ""
    
    # 显示失败任务详情
    echo "【失败任务详情】"
    ./smart-ci-metrics latest -task test-metrics-failure
    echo ""
    
    ./smart-ci-metrics stats -task test-metrics-failure
    echo ""
    
    echo "=========================================="
    echo "✅ 测试完成！"
    echo "=========================================="
else
    echo "❌ 没有找到元数据文件"
fi

# 清理
rm -f test-config.yaml
