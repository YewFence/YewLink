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

// 清理 secrets 目录中不属于当前服务列表的 .env 文件
func cleanStaleSecrets(dir string, services []string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// 目录不存在（首次运行），静默跳过
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("读取 secrets 目录 %s: %w", dir, err)
	}

	// 构建当前服务的期望文件名集合
	expected := make(map[string]struct{}, len(services))
	for _, svc := range services {
		expected[svc+".env"] = struct{}{}
	}

	var removed []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".env") {
			continue
		}
		if _, ok := expected[name]; ok {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("删除过期 secret %s: %w", path, err)
		}
		removed = append(removed, name)
	}

	if len(removed) > 0 {
		fmt.Printf("✓ 清理过期 secret: %s\n", strings.Join(removed, ", "))
	} else {
		fmt.Println("✓ 无过期 secret 文件")
	}
	return nil
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
