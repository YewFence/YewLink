package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type GeneratorConfig struct {
	projectName string
	projectPath string // 完整路径（绝对路径），用于构建符号链接命令
	outputMode  string
}

const projectHomepage = "https://github.com/YewFence/infisical-agent"

// 程序入口：读取配置、校验并渲染模板输出文件
func main() {
	var (
		servicesFile     string
		templateFile     string
		outputFile       string
		projectDir       string
		clientIDFile     string
		clientSecretFile string
	)

	flag.StringVar(&servicesFile, "services", "config.yaml", "服务配置文件路径")
	flag.StringVar(&templateFile, "template", "config.yaml.tmpl", "模板文件路径")
	flag.StringVar(&outputFile, "output", "config-no-manually-edit.yaml", "输出文件路径")
	flag.StringVar(&projectDir, "project-dir", "", "项目根目录（用于生成符号链接命令）")
	flag.StringVar(&clientIDFile, "client-id-file", "/config/client-id", "client-id 凭据文件路径（auto_discover 模式使用）")
	flag.StringVar(&clientSecretFile, "client-secret-file", "/config/client-secret", "client-secret 凭据文件路径（auto_discover 模式使用）")

	var secretsDir string
	flag.StringVar(&secretsDir, "secrets-dir", "", "secrets 输出目录（用于清理过期 .env 文件，为空则跳过清理）")
	flag.Parse()

	name, absPath := resolveProjectDir(projectDir)
	genConfig := GeneratorConfig{
		projectName: name,
		projectPath: absPath,
		outputMode:  resolveOutputMode(),
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

	// 自动发现模式：调用 Infisical API 枚举文件夹
	if config.AutoDiscover {
		clientID, err := readCredentialFile(clientIDFile)
		if err != nil {
			exitWithError("auto_discover 模式读取 client-id 失败", err)
		}
		clientSecret, err := readCredentialFile(clientSecretFile)
		if err != nil {
			exitWithError("auto_discover 模式读取 client-secret 失败", err)
		}
		token, err := fetchToken(config.Host, clientID, clientSecret)
		if err != nil {
			exitWithError("auto_discover 模式认证失败", err)
		}
		discoverPath := config.RootFolder
		if discoverPath == "" {
			discoverPath = "/"
		}
		folders, err := discoverFolders(config.Host, config.ProjectID, config.Environment, discoverPath, token)
		if err != nil {
			exitWithError("auto_discover 模式列举文件夹失败", err)
		}
		if len(folders) == 0 {
			exitWithError("auto_discover 模式未发现任何文件夹", fmt.Errorf("路径 %q 下没有子文件夹", discoverPath))
		}
		config.Services = folders
		fmt.Printf("✓ 自动发现 %d 个服务: %s\n", len(folders), strings.Join(folders, ", "))
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

	// 清理过期 secret 文件
	if secretsDir != "" {
		if err := cleanStaleSecrets(secretsDir, config.Services); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 清理过期 secret 失败: %v\n", err)
		}
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

// 统一错误输出并退出
func exitWithError(message string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", message, err)
	fmt.Fprintf(os.Stderr, "项目主页: %s\n", projectHomepage)
	os.Exit(1)
}
