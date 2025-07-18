package initialize

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDir         = "config"
	configYamlFile    = "config.yaml"
	sslDir            = "ssl"
	configYamlContent = `# QQ Bot Router 配置文件
# 这是一个用于转发QQ机器人Webhook请求的路由器配置

# ==================== 全局配置 ====================

# 日志级别配置
# 支持的级别: "development" (开发模式，详细日志) 或 "production" (生产模式，精简日志)

log_level: "development"

# 服务端口配置
# HTTPS 服务监听端口 (默认: 8443，避免需要管理员权限)
https_port: "8443"
# HTTP 服务监听端口 (用于 ACME HTTP-01 质询和重定向，生产环境建议使用 80)
http_port: "8080"

# ==================== SSL证书配置 ====================

# 域名列表 - 自动SSL证书申请
# 列出所有需要自动获取和续订 TLS 证书的域名
# 重要: 这些域名必须指向运行此服务的服务器公网IP
# 如果是本地测试，可以留空或使用测试域名
domains:
  - "your-domain.com"          # 替换为你的实际域名
  - "bot.your-domain.com"      # 可以配置多个子域名

# ==================== 机器人路由配置 ====================

# 根据域名和路径将 webhook 请求转发到不同的后端服务
# 支持多目标转发，用于备份和负载均衡
bots:
  # 示例配置 1: 简单的域名级转发
  # 适用场景: 单个机器人，所有请求转发到同一个后端
  "your-domain.com":
    # Webhook 验证密钥 (与QQ开发者平台配置保持一致)
    secret: "your-webhook-secret-here"
    # 转发目标 (支持多个地址，请求会并发转发到所有地址)
    forward_to:
      - "http://127.0.0.1:8090/webhook"     # 主要后端服务
      - "http://127.0.0.1:8091/webhook"     # 备份后端服务 (可选)

  # 示例配置 2: 基于路径的精细化路由
  # 适用场景: 多个机器人实例，根据路径分发到不同后端
  "bot.your-domain.com":
    # 默认路由配置 (当请求路径不匹配下面的具体路径时使用)
    secret: "default-webhook-secret"
    forward_to:
      - "http://127.0.0.1:9000/default"

    # 路径特定路由配置
    paths:
      # QQ频道机器人: https://bot.your-domain.com/guild
      "/guild":
        secret: "guild-bot-secret"
        forward_to:
          - "http://127.0.0.1:9001/guild-webhook"
          
      # QQ群机器人: https://bot.your-domain.com/group  
      "/group":
        secret: "group-bot-secret"
        forward_to:
          - "http://127.0.0.1:9002/group-webhook"
          - "http://127.0.0.1:9003/group-backup"   # 备份服务
          
      # 测试环境: https://bot.your-domain.com/test
      "/test":
        secret: "test-secret"
        forward_to:
          - "http://127.0.0.1:9999/test-webhook"

# ==================== 配置说明 ====================
# 
# 1. 首次运行时，程序会自动创建此配置文件
# 2. 修改配置后需要重启服务生效 (热重载功能开发中)
# 3. secret 必须与QQ开发者平台的Webhook密钥一致
# 4. forward_to 支持多个目标，用于实现高可用和负载均衡
# 5. 转发失败不会影响其他目标，错误日志级别为 DEBUG
# 6. 生产环境建议使用 log_level: "production"
# 7. SSL证书自动存储在 secret-dir/ 目录下
#
# 更多信息请参考项目文档
`
)

// CheckConfig 检查配置目录和文件是否存在，如果不存在则创建它们
func CheckConfig() error {
	// 检查 config 目录
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		fmt.Println("Config directory not found, creating...")
		if err := os.Mkdir(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// 检查 config.yaml
	configYamlPath := filepath.Join(configDir, configYamlFile)
	if _, err := os.Stat(configYamlPath); os.IsNotExist(err) {
		fmt.Println("config.yaml not found, creating template...")
		if err := os.WriteFile(configYamlPath, []byte(configYamlContent), 0644); err != nil {
			return fmt.Errorf("failed to create config.yaml: %w", err)
		}
	}

	// 检查 config/ssl 目录
	sslPath := filepath.Join(configDir, sslDir)
	if _, err := os.Stat(sslPath); os.IsNotExist(err) {
		fmt.Println("SSL directory not found, creating...")
		if err := os.Mkdir(sslPath, 0755); err != nil {
			return fmt.Errorf("failed to create ssl directory: %w", err)
		}
	}

	return nil
}
