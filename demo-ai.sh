#!/bin/bash

# SmartCI AI集成功能演示脚本

set -e

echo "================================================"
echo "SmartCI AI集成功能演示"
echo "================================================"
echo ""

# 创建测试配置文件
cat > config-ai-demo.yaml << 'EOF'
server:
  host: "localhost"
  port: 8080

llm_key: "demo-key"
llm_base: "https://api.openai.com/v1"

schedule: "@every 1h"

bash_tasks:
  # 成功的任务
  - name: "success-task"
    description: "一个会成功的任务"
    command: |
      echo "开始执行任务..."
      echo "处理中..."
      sleep 1
      echo "任务完成！"
    ai:
      enabled: true
      context:
        - "log"
      prompt: "分析任务执行情况，总结结果"
      output_file: "ai-report.md"

  # 失败的任务
  - name: "failure-task"
    description: "一个会失败的任务"
    command: |
      echo "开始执行任务..."
      echo "ERROR: 发生错误！"
      exit 1
    ai:
      enabled: true
      context:
        - "log"
      prompt: "分析任务失败原因，给出修复建议"
      output_file: "failure-analysis.md"

  # 带多个上下文的任务
  - name: "multi-context-task"
    description: "测试多上下文收集"
    command: |
      # 创建测试文件
      mkdir -p reports coverage
      echo '{"tests": 10, "passed": 8, "failed": 2}' > reports/test-results.json
      echo '{"coverage": "85%"}' > coverage/report.json
      echo "详细日志内容..." > detailed.log
      echo "任务完成"
    ai:
      enabled: true
      context:
        - "log"
        - "*.log"
        - "reports/**/*.json"
        - "coverage/**/*.json"
      prompt: "综合分析测试结果、代码覆盖率和日志，给出详细报告"
      output_file: "comprehensive-report.md"
EOF

echo "✅ 创建演示配置文件: config-ai-demo.yaml"
echo ""

# 构建项目
echo "📦 构建SmartCI..."
go build -o smart-ci . || {
    echo "❌ 构建失败"
    exit 1
}
echo "✅ 构建成功"
echo ""

# 清理旧的日志
echo "🧹 清理旧日志..."
rm -rf logs/*
mkdir -p logs
echo "✅ 清理完成"
echo ""

# 启动服务器（后台运行）
echo "🚀 启动SmartCI服务器..."
./smart-ci -config config-ai-demo.yaml > server.log 2>&1 &
SERVER_PID=$!
echo "✅ 服务器已启动 (PID: $SERVER_PID)"
echo ""

# 等待服务器启动
sleep 2

# 测试1：成功的任务
echo "================================================"
echo "测试 1: 执行成功的任务"
echo "================================================"
curl -s "http://localhost:8080/webhook/bash?task=success-task" || true
sleep 2

# 查找最新的任务目录
LATEST_DIR=$(ls -td logs/*/ 2>/dev/null | head -1)
if [ -n "$LATEST_DIR" ]; then
    echo ""
    echo "📁 任务目录: $LATEST_DIR"
    echo ""
    echo "📄 任务日志:"
    cat "${LATEST_DIR}task.log" || true
    echo ""
    echo "🤖 AI分析报告:"
    if [ -f "${LATEST_DIR}ai-report.md" ]; then
        cat "${LATEST_DIR}ai-report.md"
    else
        echo "（AI分析报告尚未生成或AI功能未实现）"
    fi
    echo ""
fi

# 测试2：失败的任务
echo "================================================"
echo "测试 2: 执行失败的任务"
echo "================================================"
curl -s "http://localhost:8080/webhook/bash?task=failure-task" || true
sleep 2

# 查找最新的任务目录
LATEST_DIR=$(ls -td logs/*/ 2>/dev/null | head -1)
if [ -n "$LATEST_DIR" ]; then
    echo ""
    echo "📁 任务目录: $LATEST_DIR"
    echo ""
    echo "📄 任务日志:"
    cat "${LATEST_DIR}task.log" || true
    echo ""
    echo "🤖 AI失败分析:"
    if [ -f "${LATEST_DIR}failure-analysis.md" ]; then
        cat "${LATEST_DIR}failure-analysis.md"
    else
        echo "（AI分析报告尚未生成或AI功能未实现）"
    fi
    echo ""
fi

# 测试3：多上下文任务
echo "================================================"
echo "测试 3: 执行多上下文任务"
echo "================================================"
curl -s "http://localhost:8080/webhook/bash?task=multi-context-task" || true
sleep 3

# 查找最新的任务目录
LATEST_DIR=$(ls -td logs/*/ 2>/dev/null | head -1)
if [ -n "$LATEST_DIR" ]; then
    echo ""
    echo "📁 任务目录: $LATEST_DIR"
    echo ""
    echo "📂 目录结构:"
    tree "$LATEST_DIR" 2>/dev/null || ls -lR "$LATEST_DIR"
    echo ""
    echo "🤖 综合AI报告:"
    if [ -f "${LATEST_DIR}comprehensive-report.md" ]; then
        cat "${LATEST_DIR}comprehensive-report.md"
    else
        echo "（AI分析报告尚未生成或AI功能未实现）"
    fi
    echo ""
fi

# 展示所有任务目录
echo "================================================"
echo "所有任务目录一览"
echo "================================================"
echo ""
for dir in logs/*/; do
    if [ -d "$dir" ]; then
        task_id=$(basename "$dir")
        echo "📁 任务ID: $task_id"
        if [ -f "${dir}task.log" ]; then
            echo "   📄 日志: ${dir}task.log"
        fi
        # 列出所有AI报告文件
        for report in "${dir}"*.md; do
            if [ -f "$report" ]; then
                echo "   🤖 报告: $(basename "$report")"
            fi
        done
        echo ""
    fi
done

# 停止服务器
echo "================================================"
echo "清理"
echo "================================================"
echo "🛑 停止服务器..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo "✅ 服务器已停止"
echo ""

echo "================================================"
echo "演示完成！"
echo "================================================"
echo ""
echo "💡 提示："
echo "  - 所有任务数据保存在 logs/ 目录下"
echo "  - 每个任务都有唯一的ID作为目录名"
echo "  - AI分析报告保存在任务目录中"
echo "  - 可以通过配置文件自定义上下文和Prompt"
echo ""
echo "📚 更多文档请查看: docs/ai-integration.md"
