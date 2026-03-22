package main

import (
	"bytes"
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
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
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
	// 规范化并校验服务名称
	if err := validateServices(config.Services); err != nil {
		return err
	}
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

// validateServiceName 校验单个服务名是否安全可用于文件路径和 shell 命令
func validateServiceName(name string) error {
	if name == "" || name == "." {
		return fmt.Errorf("服务名为空或无效")
	}
	if name != strings.TrimSpace(name) {
		return fmt.Errorf("服务名 %q 包含首尾空白字符", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("服务名 %q 包含路径分隔符", name)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("服务名 %q 包含路径穿越序列", name)
	}
	return nil
}

// validateServices 对服务名列表进行 trim + 逐项校验，原地修改 slice
func validateServices(services []string) error {
	for i, svc := range services {
		services[i] = strings.TrimSpace(svc)
		if err := validateServiceName(services[i]); err != nil {
			return err
		}
	}
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
