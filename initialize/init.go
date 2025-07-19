package initialize

import (
	"fmt"
	"os"
)

const (
	configYamlFile = "config.yaml"
	sslDir         = "ssl"
	secretDir      = "secret-dir"
)

// CheckConfig 检查并创建必要的目录结构
func CheckConfig() error {
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

	fmt.Println("Directory initialization completed successfully.")
	return nil
}
