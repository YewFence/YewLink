package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host            string   `yaml:"host"`
	ProjectID       string   `yaml:"project_id"`
	Environment     string   `yaml:"environment"`
	PollingInterval string   `yaml:"polling_interval"`
	RootFolder      string   `yaml:"root_folder"`
	Services        []string `yaml:"services"`
}

type GeneratorConfig struct {
	projectDir string
	outputMode string
}

const (
	outputModeRelative = "relative"
	outputModeAbsolute = "absolute"
)

const projectHomepage = "https://github.com/YewFence/infisical-agent"

// 解析项目目录：优先使用传入的参数，其次从环境变量，最后自动检测
// 如果是绝对路径，提取最后一个目录名
func resolveProjectDir(flagValue string) string {
	if flagValue != "" {
		return extractDirName(flagValue)
	}
	// 从环境变量读取 PROJECT_DIR
	if envValue := os.Getenv("PROJECT_DIR"); envValue != "" {
		return extractDirName(envValue)
	}
	// 自动检测：优先用可执行文件所在目录，其次用工作目录
	if name := getExecutableDirName(); name != "" && name != "." {
		return name
	}
	return getWorkingDirName()
}

// 从路径中提取最后一个目录名（无论绝对或相对路径）
func extractDirName(path string) string {
	return filepath.Base(path)
}

// 解析输出模式：从环境变量读取，默认相对路径
func resolveOutputMode() string {
	mode := os.Getenv("OUTPUT_MODE")
	switch strings.ToLower(mode) {
	case outputModeAbsolute:
		return outputModeAbsolute
	default:
		return outputModeRelative
	}
}

// 构建符号链接命令
func buildLnCommand(service string, cfg GeneratorConfig) string {
	if cfg.outputMode == outputModeAbsolute {
		// 绝对路径模式：直接 ln 到完整路径
		secretsPath := filepath.Join(cfg.projectDir, "secrets", service+".env")
		targetPath := filepath.Join(cfg.projectDir, service, ".env")
		return fmt.Sprintf("ln -sf %s %s", secretsPath, targetPath)
	}

	// 相对路径模式：secrets 相对于服务目录的路径
	// cd 到 ../<service> 后，secrets 路径是 ../<projectDir>/secrets/<service>.env
	relSecretsPath := filepath.Join("..", service, "secrets", service+".env")
	return fmt.Sprintf("cd ../%s && ln -sf %s .env", service, relSecretsPath)
}

// 构建备份命令
func buildMvCommand(service string, cfg GeneratorConfig) string {
	if cfg.outputMode == outputModeAbsolute {
		// 绝对路径模式
		targetPath := filepath.Join(cfg.projectDir, service, ".env")
		return fmt.Sprintf("mv %s %s.bak", targetPath, targetPath)
	}

	// 相对路径模式
	return fmt.Sprintf("cd ../%s && mv .env .env.bak", service)
}

// 获取可执行文件所在目录名
func getExecutableDirName() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	if dir == "." || dir == "" {
		return ""
	}
	return filepath.Base(dir)
}

// 获取当前工作目录名作为回退
func getWorkingDirName() string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" || cwd == "." {
		return "infisical-agent"
	}
	return filepath.Base(cwd)
}

// 程序入口：读取配置、校验并渲染模板输出文件
func main() {
	var (
		servicesFile string
		templateFile string
		outputFile   string
		projectDir   string
	)

	flag.StringVar(&servicesFile, "services", "config.yaml", "服务配置文件路径")
	flag.StringVar(&templateFile, "template", "config.yaml.tmpl", "模板文件路径")
	flag.StringVar(&outputFile, "output", "config-no-manually-edit.yaml", "输出文件路径")
	flag.StringVar(&projectDir, "project-dir", "", "项目根目录（用于生成符号链接命令）")
	flag.Parse()

	genConfig := GeneratorConfig{
		projectDir: resolveProjectDir(projectDir),
		outputMode: resolveOutputMode(),
	}

	// 读取服务配置
	config, err := loadConfig(servicesFile)
	if err != nil {
		exitWithError("读取配置失败", err)
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		exitWithError("配置验证失败", err)
	}

	// 加载模板
	tmpl, err := template.New(filepath.Base(templateFile)).Funcs(template.FuncMap{
		"secretPath": buildSecretPath,
	}).ParseFiles(templateFile)
	if err != nil {
		exitWithError("加载模板失败", err)
	}

	// 生成输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		exitWithError("创建输出文件失败", err)
	}
	defer outFile.Close()

	if err := tmpl.Execute(outFile, config); err != nil {
		exitWithError("渲染模板失败", err)
	}

	absOutput, _ := filepath.Abs(outputFile)
	fmt.Printf("✓ 已生成配置文件: %s\n", absOutput)
	fmt.Printf("  - 项目 ID: %s\n", config.ProjectID)
	fmt.Printf("  - 环境: %s\n", config.Environment)
	if config.RootFolder != "" {
		fmt.Printf("  - 根文件夹: %s\n", config.RootFolder)
	} else {
		fmt.Printf("  - 根文件夹: (无)\n")
	}
	fmt.Printf("  - 服务数量: %d\n", len(config.Services))
	for _, svc := range config.Services {
		fmt.Printf("    • %s\n", svc)
	}

	// 打印符号链接命令供复制
	fmt.Println("\n📋 在各服务目录下创建符号链接:")
	for _, svc := range config.Services {
		cmd := buildLnCommand(svc, genConfig)
		fmt.Printf("    %s\n", cmd)
	}

	// 打印 env_file 路径供复制
	fmt.Println("\n📋 同时在 docker-compose.yml 中添加 env_file:")
	fmt.Println("    env_file: .env")

	// 打印备份建议
	fmt.Printf("\n💡 建议先备份原 .env 文件（如果有）\n")
	for _, svc := range config.Services {
		cmd := buildMvCommand(svc, genConfig)
		fmt.Printf("    %s\n", cmd)
	}

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
	if len(config.Services) == 0 {
		return fmt.Errorf("请在 config.yaml 中至少添加一个服务")
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

// 拼接根路径与服务名，生成密钥路径
func buildSecretPath(root, service string) string {
	service = strings.TrimSpace(service)
	service = strings.Trim(service, "/")
	if service == "" {
		return root
	}
	if root == "" {
		return "/" + service
	}
	return root + "/" + service
}

// 统一错误输出并退出
func exitWithError(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	fmt.Fprintf(os.Stderr, "项目主页: %s\n", projectHomepage)
	os.Exit(1)
}
