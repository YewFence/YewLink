package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	outputModeRelative = "relative"
	outputModeAbsolute = "absolute"
)

// 解析项目目录：返回目录名和完整路径
// 优先使用传入的参数，其次从环境变量，最后自动检测
func resolveProjectDir(flagValue string) (name string, absPath string) {
	if flagValue != "" {
		return filepath.Base(flagValue), flagValue
	}
	// 从环境变量读取 PROJECT_DIR
	if envValue := os.Getenv("PROJECT_DIR"); envValue != "" {
		return filepath.Base(envValue), envValue
	}
	// 自动检测：优先用可执行文件所在目录，其次用工作目录
	if exePath := getExecutableDir(); exePath != "" && exePath != "." {
		return filepath.Base(exePath), exePath
	}
	cwd, _ := os.Getwd()
	name = filepath.Base(cwd)
	if name == "" || name == "." {
		name = "infisical-agent"
	}
	return name, cwd
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
	secretsPath := filepath.Join(cfg.projectPath, "secrets", service+".env")

	if cfg.outputMode == outputModeAbsolute {
		// 绝对路径模式
		targetPath := filepath.Join(cfg.projectPath, service, ".env")
		return fmt.Sprintf("ln -sf %s %s", secretsPath, targetPath)
	}

	// 相对路径模式：服务目录在 ../<service>（与 projectPath 同级）
	// 从服务目录看，secrets 的相对路径是 ../<projectName>/secrets/<service>.env
	relSecretsPath := filepath.Join("..", cfg.projectName, "secrets", service+".env")
	return fmt.Sprintf("cd ../%s && ln -sf %s .env", service, relSecretsPath)
}

// 构建备份命令
func buildMvCommand(service string, cfg GeneratorConfig) string {
	if cfg.outputMode == outputModeAbsolute {
		// 绝对路径模式
		targetPath := filepath.Join(cfg.projectPath, service, ".env")
		return fmt.Sprintf("mv %s %s.bak", targetPath, targetPath)
	}

	// 相对路径模式
	return fmt.Sprintf("cd ../%s && mv .env .env.bak", service)
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

// 获取可执行文件所在目录（完整路径）
func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	dir := filepath.Dir(exe)
	if dir == "." || dir == "" {
		return ""
	}
	return dir
}

// 获取可执行文件所在目录名
func getExecutableDirName() string {
	return filepath.Base(getExecutableDir())
}

// 获取当前工作目录名作为回退
func getWorkingDirName() string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" || cwd == "." {
		return "infisical-agent"
	}
	return filepath.Base(cwd)
}
