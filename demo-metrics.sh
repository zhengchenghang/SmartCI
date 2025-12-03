#!/bin/bash

echo "=========================================="
echo "SmartCI Metrics 功能演示"
echo "=========================================="
echo ""

# 检查是否已构建
if [ ! -f "./smart-ci-metrics" ]; then
    echo "📦 构建 metrics 工具..."
    go build -o smart-ci-metrics ./cmd/metrics/main.go
    if [ $? -ne 0 ]; then
        echo "❌ 构建失败"
        exit 1
    fi
    echo "✅ 构建成功"
    echo ""
fi

# 检查日志目录
if [ ! -d "./logs" ]; then
    echo "❌ 日志目录不存在，请先运行一些任务"
    exit 1
fi

# 检查是否有任务记录
TASK_COUNT=$(find ./logs -name "metadata.json" 2>/dev/null | wc -l)
if [ "$TASK_COUNT" -eq 0 ]; then
    echo "📭 没有找到任务执行记录"
    echo ""
    echo "💡 提示：请先执行一些任务以生成数据"
    echo "   例如：curl 'http://localhost:8080/webhook/bash?task=<task_name>'"
    echo ""
    exit 0
fi

echo "📊 找到 $TASK_COUNT 条任务执行记录"
echo ""

# 显示所有任务概览
echo "=========================================="
echo "1. 所有任务概览"
echo "=========================================="
./smart-ci-metrics all
echo ""

# 获取第一个任务名称作为示例
FIRST_TASK=$(find ./logs -name "metadata.json" -print0 | xargs -0 cat | grep -o '"task_name":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -n "$FIRST_TASK" ]; then
    echo "=========================================="
    echo "2. 任务详情示例: $FIRST_TASK"
    echo "=========================================="
    
    # 最近一次执行
    echo ""
    echo "【最近一次执行】"
    ./smart-ci-metrics latest -task "$FIRST_TASK"
    echo ""
    
    # 执行历史
    echo "【执行历史（最近10条）】"
    ./smart-ci-metrics list -task "$FIRST_TASK" -limit 10
    echo ""
    
    # 统计信息
    echo "【统计信息】"
    ./smart-ci-metrics stats -task "$FIRST_TASK"
    echo ""
fi

echo "=========================================="
echo "3. 使用示例"
echo "=========================================="
echo ""
echo "查看帮助："
echo "  ./smart-ci-metrics"
echo ""
echo "查看最近一次执行："
echo "  ./smart-ci-metrics latest -task <task_name>"
echo ""
echo "查看最近7天的执行记录："
echo "  ./smart-ci-metrics list -task <task_name> -days 7"
echo ""
echo "查看统计信息："
echo "  ./smart-ci-metrics stats -task <task_name> -days 30"
echo ""
echo "查看所有任务："
echo "  ./smart-ci-metrics all"
echo ""
echo "=========================================="
echo "✅ 演示完成"
echo "=========================================="
