#!/bin/bash

# 数据库实例创建测试脚本

echo "========================================="
echo "数据库实例创建测试"
echo "========================================="

# 配置
API_BASE="http://localhost:12306/api/v1"

# 步骤1: 登录获取token
echo ""
echo "步骤1: 登录获取JWT token..."
LOGIN_RESPONSE=$(curl -s -X POST ${API_BASE}/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

echo "登录响应: $LOGIN_RESPONSE"

# 提取token
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败，无法获取token"
    echo "请检查:"
    echo "1. 应用是否正在运行: ps aux | grep tiga"
    echo "2. 默认管理员账号是否存在"
    echo "3. 或者创建新用户后使用正确的凭据"
    exit 1
fi

echo "✅ 成功获取token: ${TOKEN:0:20}..."

# 步骤2: 创建MySQL实例
echo ""
echo "步骤2: 创建MySQL测试实例..."
CREATE_RESPONSE=$(curl -s -X POST ${API_BASE}/database/instances \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "MySQL Test Instance",
    "type": "mysql",
    "host": "localhost",
    "port": 3306,
    "username": "root",
    "password": "your-password-here",
    "description": "MySQL测试实例"
  }')

echo "创建响应: $CREATE_RESPONSE"

# 检查是否成功
if echo "$CREATE_RESPONSE" | grep -q '"id"'; then
    echo "✅ 实例创建成功！"
    INSTANCE_ID=$(echo $CREATE_RESPONSE | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "实例ID: $INSTANCE_ID"
else
    echo "❌ 实例创建失败"
    echo "可能的原因:"
    echo "1. MySQL服务未运行或连接失败"
    echo "2. 用户名密码不正确"
    echo "3. 网络连接问题"
    exit 1
fi

# 步骤3: 列出所有实例
echo ""
echo "步骤3: 列出所有数据库实例..."
LIST_RESPONSE=$(curl -s -X GET ${API_BASE}/database/instances \
  -H "Authorization: Bearer $TOKEN")

echo "实例列表: $LIST_RESPONSE"

# 步骤4: 测试连接
if [ ! -z "$INSTANCE_ID" ]; then
    echo ""
    echo "步骤4: 测试实例连接..."
    TEST_RESPONSE=$(curl -s -X POST ${API_BASE}/database/instances/${INSTANCE_ID}/test \
      -H "Authorization: Bearer $TOKEN")

    echo "连接测试响应: $TEST_RESPONSE"
fi

echo ""
echo "========================================="
echo "测试完成"
echo "========================================="
