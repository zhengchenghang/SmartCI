#!/bin/bash

# 应用部署脚本
# 使用方法: ./deploy.sh [环境]

set -e  # 遇到错误立即退出

ENVIRONMENT=${1:-production}
echo "开始部署应用到 $ENVIRONMENT 环境..."

# 检查必要的环境变量
if [ -z "$DEPLOY_USER" ]; then
    echo "错误: 未设置 DEPLOY_USER 环境变量"
    exit 1
fi

if [ -z "$DEPLOY_HOST" ]; then
    echo "错误: 未设置 DEPLOY_HOST 环境变量"
    exit 1
fi

# 1. 备份当前版本
echo "备份当前版本..."
ssh $DEPLOY_USER@$DEPLOY_HOST "cd /opt/app && cp -r current current_backup_$(date +%Y%m%d_%H%M%S)"

# 2. 拉取最新代码
echo "拉取最新代码..."
git pull origin main

# 3. 构建应用
echo "构建应用..."
if [ -f "package.json" ]; then
    npm ci --production
    npm run build
elif [ -f "go.mod" ]; then
    go build -o app main.go
elif [ -f "requirements.txt" ]; then
    pip install -r requirements.txt
fi

# 4. 运行测试
echo "运行测试..."
if [ -f "package.json" ]; then
    npm test
elif [ -f "go.mod" ]; then
    go test ./...
elif [ -f "requirements.txt" ]; then
    python -m pytest
fi

# 5. 部署到服务器
echo "部署到服务器..."
rsync -av --delete ./ $DEPLOY_USER@$DEPLOY_HOST:/opt/app/new/

# 6. 重启服务
echo "重启服务..."
ssh $DEPLOY_USER@$DEPLOY_HOST "
    cd /opt/app
    mv new current
    sudo systemctl restart app
    sleep 5
    if systemctl is-active --quiet app; then
        echo '应用重启成功'
    else
        echo '应用重启失败，回滚...'
        mv current_backup_$(date +%Y%m%d_%H%M%S) current
        sudo systemctl restart app
        exit 1
    fi
"

# 7. 健康检查
echo "执行健康检查..."
sleep 10
curl -f http://$DEPLOY_HOST:3000/health || {
    echo "健康检查失败"
    exit 1
}

echo "部署完成！应用已成功部署到 $ENVIRONMENT 环境"