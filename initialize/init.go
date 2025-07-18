package initialize

import (
	"fmt"
	"os"
)

const (
	configYamlFile    = "config.yaml"
	sslDir            = "ssl"
	secretDir         = "secret-dir"
	configYamlContent = `# QQ Bot Router Configuration
# 新版本配置文件 - 基于完整 webhook URL 的路由配置

# 日志级别: production, development
log_level: development

# 服务端口配置
https_port: "8443"    
http_port: "8444"

# 机器人配置 - 使用完整 webhook URL 作为键
# 系统会自动从配置中提取域名用于 SSL 证书管理
bots:
  # 示例 1: 简单的 webhook URL
  "your-domain.com/webhook":
    secret: "your-bot-secret-here"
    forward_to:
      - "http://localhost:3000/webhook"
      - "http://localhost:3001/webhook"    # mirror 
  
  # 示例 2: 带端口的 webhook URL
  "bot.your-domain.com:8443/api/webhook":
    secret: "another-secret"
    forward_to:
      - "http://outside.site:8080/qq-events"
  
  # 示例 3: 不同域名的自定义路径
  "test.example.com/bot/events":
    secret: "third-secret"
    forward_to:
      - "http://127.0.0.1:5000/handle-qq"
      - "http://backup-server:9000/qq"

# ==================== 配置说明 ====================
# 
# 1. 系统自动从 bot 配置中提取唯一域名用于 SSL 证书管理
# 2. 每个 bot 配置将完整的 webhook URL 映射到转发目标
# 3. 支持多个 forward_to 目标，实现负载均衡和冗余
# 4. secret 用于 webhook 签名验证
# 5. 配置变更会被自动检测并重新加载
# 6. 转发失败时仍返回成功 ACK，防止平台重试导致消息风暴
# 7. SSL 证书自动存储在 secret-dir/ 目录下
#
# 更多信息请参考项目文档(但是并没有文档)
`
)

// CheckConfig 检查并创建配置文件和目录结构
func CheckConfig() error {
	// 检查并创建 config.yaml 文件（在项目根目录）
	if _, err := os.Stat(configYamlFile); os.IsNotExist(err) {
		fmt.Println("config.yaml not found, creating from template...")
		if err := os.WriteFile(configYamlFile, []byte(configYamlContent), 0644); err != nil {
			return fmt.Errorf("failed to create config.yaml: %w", err)
		}
		fmt.Printf("Created config file: %s\n", configYamlFile)
	}

	// 检查并创建 ssl 目录（在项目根目录）
	if _, err := os.Stat(sslDir); os.IsNotExist(err) {
		fmt.Println("SSL directory not found, creating...")
		if err := os.MkdirAll(sslDir, 0755); err != nil {
			// 这不是致命错误，只记录警告
			fmt.Printf("Warning: failed to create SSL directory: %v\n", err)
		} else {
			fmt.Printf("Created SSL directory: %s\n", sslDir)
		}
	}

	// 检查并创建 secret-dir 目录（在项目根目录）
	if _, err := os.Stat(secretDir); os.IsNotExist(err) {
		fmt.Printf("Secret directory not found, creating: %s\n", secretDir)
		if err := os.MkdirAll(secretDir, 0755); err != nil {
			// 这不是致命错误，只记录警告
			fmt.Printf("Warning: failed to create secret directory: %v\n", err)
		} else {
			fmt.Printf("Created secret directory: %s\n", secretDir)
		}
	}

	fmt.Println("Configuration initialization completed successfully.")
	return nil
}
