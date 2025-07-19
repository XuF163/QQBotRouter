package initialize

import (
	"fmt"
	"os"
)

const (
	configYamlFile    = "config.yaml"
	sslDir            = "ssl"
	secretDir         = "secret-dir"
	configYamlContent = `# QQ Bot Router Configuration with Intelligent QoS
# 智能QoS升级版配置文件

# 日志级别: production, development
log_level: development

# 服务端口配置
https_port: "8443"
http_port: "8444"

# 机器人配置
bots:
  # 示例配置 - 支持正则路由和普通转发
  "your-domain.com/webhook":
    secret: "your-bot-secret-here"
    forward_to:
      - "http://localhost:3000/webhook"
      - "http://localhost:3001/webhook"
    
    # 正则路由配置 - 根据消息内容智能路由
    regex_routes:
      "^#帮助":
        urls:
          - "http://localhost:3000/help"
          - "http://localhost:3001/help"
      "^#测试":
        endpoints:
          - "http://localhost:3000/test"

# 智能调度策略配置 (QoS)
intelligent_scheduling_policy:
  enabled: true
  
  # 动态负载调节
  dynamic_load_tuning:
    enabled: true
    latency_threshold: 250  # 延迟阈值(ms)
    
  # 自适应节流
  adaptive_throttling:
    enabled: true
    min_request_interval: 100  # 最小请求间隔(ms)
    
  # 认知调度
  cognitive_scheduling:
    worker_pool_size: 16
    model_retrain_interval: 24  # 小时
    fast_user_sensitivity: 1.5
    spam_user_sensitivity: 3.0
    
  # 动态基线分析
  dynamic_baseline_analysis:
    enabled: true
    initial_learning_duration: 60  # 分钟
    pattern_recognition_sensitivity: 2.0
    min_data_points_for_baseline: 100
    
  # 热重载配置
  hot_reload:
    enabled: true
    config_watch_interval: 10  # 秒

# ==================== 智能QoS功能说明 ====================
# 
# 1. 动态负载调节: 根据系统负载自动调整请求处理策略
# 2. 自适应节流: 智能识别并限制异常请求频率
# 3. 认知调度: 基于用户行为模式的智能调度
# 4. 动态基线分析: 自动学习正常流量模式
# 5. 正则路由: 支持基于消息内容的智能路由
# 6. 热重载: 配置变更自动生效，无需重启
# 7. SSL证书自动管理: 自动申请和续期Let's Encrypt证书
#
# 更多功能特性请参考项目文档
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
