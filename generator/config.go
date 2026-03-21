package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host            string   `yaml:"host"`
	ProjectID       string   `yaml:"project_id"`
	Environment     string   `yaml:"environment"`
	PollingInterval string   `yaml:"polling_interval"`
	RootFolder      string   `yaml:"root_folder"`
	Services        []string `yaml:"services"`
	AutoDiscover    bool     `yaml:"auto_discover"`
}

// 读取并解析 YAML 配置文件
func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取文件 %s: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析 YAML: %w", err)
	}

	return &config, nil
}

// 校验配置并填充默认值
func validateConfig(config *Config) error {
	if config.ProjectID == "" || config.ProjectID == "<your-project-id>" {
		return fmt.Errorf("请在 config.yaml 中设置有效的 project_id")
	}
	if config.Environment == "" {
		return fmt.Errorf("请在 config.yaml 中设置 environment")
	}
	if !config.AutoDiscover && len(config.Services) == 0 {
		return fmt.Errorf("请在 config.yaml 中至少添加一个服务，或设置 auto_discover: true")
	}
	// 规范化服务名称：去掉首尾空白并确保每项非空
	trimmedServices := make([]string, 0, len(config.Services))
	for _, service := range config.Services {
		trimmed := strings.TrimSpace(service)
		if trimmed == "" {
			return fmt.Errorf("请在 config.yaml 中为每个服务设置有效的名称")
		}
		trimmedServices = append(trimmedServices, trimmed)
	}
	config.Services = trimmedServices
	if config.PollingInterval == "" {
		config.PollingInterval = "300s"
	}
	if config.Host == "" {
		config.Host = "https://app.infisical.com"
	}
	config.Host = strings.TrimRight(config.Host, "/")
	config.RootFolder = normalizeRootFolder(config.RootFolder)
	return nil
}

// 规范化根路径，确保以单个 "/" 开头且无多余分隔符
func normalizeRootFolder(root string) string {
	root = strings.TrimSpace(root)
	root = strings.Trim(root, "/")
	if root == "" {
		return ""
	}
	return "/" + root
}
