# SmartCI Makefile

.PHONY: build build-server build-client build-metrics clean test run-server help

# 默认目标
all: build

# 构建所有可执行文件
build: build-server build-client build-metrics

# 构建服务器
build-server:
	@echo "🔨 构建服务器..."
	go build -o smart-ci-server main.go

# 构建客户端
build-client:
	@echo "🔨 构建客户端..."
	go build -o smart-ci-client ./client/main.go

# 构建metrics命令行工具
build-metrics:
	@echo "🔨 构建metrics工具..."
	go build -o smart-ci-metrics ./cmd/metrics/main.go

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	rm -f smart-ci-server smart-ci-client smart-ci-metrics

# 运行测试
test:
	@echo "🧪 运行测试..."
	go test ./...

# 运行服务器
run-server: build-server
	@echo "🚀 启动SmartCI服务器..."
	./smart-ci-server -mode server -config config.yaml

# 构建并运行服务器
dev: build-server
	@echo "🔧 开发模式启动服务器..."
	./smart-ci-server -mode server -config config.yaml -host localhost -port 8080

# 安装到系统路径
install: build
	@echo "📦 安装到 /usr/local/bin..."
	sudo cp smart-ci-server /usr/local/bin/
	sudo cp smart-ci-client /usr/local/bin/
	sudo cp smart-ci-metrics /usr/local/bin/

# 显示帮助
help:
	@echo "SmartCI 构建工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build          - 构建服务器、客户端和metrics工具"
	@echo "  build-server   - 只构建服务器"
	@echo "  build-client   - 只构建客户端"
	@echo "  build-metrics  - 只构建metrics工具"
	@echo "  clean          - 清理构建文件"
	@echo "  test           - 运行测试"
	@echo "  run-server     - 构建并运行服务器"
	@echo "  dev            - 开发模式启动服务器"
	@echo "  install        - 安装到系统路径"
	@echo "  help           - 显示此帮助信息"
	@echo ""
	@echo "使用示例:"
	@echo "  make build                          # 构建所有"
	@echo "  make run-server                     # 启动服务器"
	@echo "  ./smart-ci-metrics latest -task xxx # 查看任务最近执行"
	@echo "  ./smart-ci-metrics stats -task xxx  # 查看任务统计"
