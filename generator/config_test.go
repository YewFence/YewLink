package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigSuccess(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := strings.Join([]string{
		`host: "https://self-hosted.infisical.test"`,
		`project_id: "project-123"`,
		`environment: "prod"`,
		`polling_interval: "60s"`,
		`root_folder: "/team-a"`,
		"services:",
		`  - " nginx "`,
	}, "\n")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	config, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if config.Host != "https://self-hosted.infisical.test" {
		t.Fatalf("Host = %q, want %q", config.Host, "https://self-hosted.infisical.test")
	}
	if config.ProjectID != "project-123" {
		t.Fatalf("ProjectID = %q, want %q", config.ProjectID, "project-123")
	}
	if config.Environment != "prod" {
		t.Fatalf("Environment = %q, want %q", config.Environment, "prod")
	}
	if config.PollingInterval != "60s" {
		t.Fatalf("PollingInterval = %q, want %q", config.PollingInterval, "60s")
	}
	if config.RootFolder != "/team-a" {
		t.Fatalf("RootFolder = %q, want %q", config.RootFolder, "/team-a")
	}
	if len(config.Services) != 1 || config.Services[0] != " nginx " {
		t.Fatalf("Services = %#v, want original YAML values", config.Services)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		path    string
		content string
	}{
		{
			name: "missing file",
			path: filepath.Join(t.TempDir(), "missing.yaml"),
		},
		{
			name:    "invalid yaml",
			path:    filepath.Join(t.TempDir(), "invalid.yaml"),
			content: "services: [unterminated",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if tc.content != "" {
				if err := os.WriteFile(tc.path, []byte(tc.content), 0o600); err != nil {
					t.Fatalf("write config: %v", err)
				}
			}

			if _, err := loadConfig(tc.path); err == nil {
				t.Fatalf("loadConfig(%q) expected error", tc.path)
			}
		})
	}
}

func TestValidateConfigDefaultsAndNormalization(t *testing.T) {
	t.Helper()

	config := &Config{
		ProjectID:   "project-123",
		Environment: "prod",
		RootFolder:  " //team-a/// ",
		Services:    []string{" nginx ", "api"},
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}

	if config.PollingInterval != "300s" {
		t.Fatalf("PollingInterval = %q, want %q", config.PollingInterval, "300s")
	}
	if config.Host != "https://app.infisical.com" {
		t.Fatalf("Host = %q, want %q", config.Host, "https://app.infisical.com")
	}
	if config.RootFolder != "/team-a" {
		t.Fatalf("RootFolder = %q, want %q", config.RootFolder, "/team-a")
	}
	if got, want := config.Services, []string{"nginx", "api"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Services = %#v, want %#v", got, want)
	}
}

func TestValidateConfigErrors(t *testing.T) {
	t.Helper()

	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name: "missing project id",
			config: Config{
				Environment: "prod",
				Services:    []string{"nginx"},
			},
			want: "project_id",
		},
		{
			name: "placeholder project id",
			config: Config{
				ProjectID:   "<your-project-id>",
				Environment: "prod",
				Services:    []string{"nginx"},
			},
			want: "project_id",
		},
		{
			name: "missing environment",
			config: Config{
				ProjectID: "project-123",
				Services:  []string{"nginx"},
			},
			want: "environment",
		},
		{
			name: "missing services without auto discover",
			config: Config{
				ProjectID:   "project-123",
				Environment: "prod",
			},
			want: "auto_discover",
		},
		{
			name: "empty trimmed service name",
			config: Config{
				ProjectID:   "project-123",
				Environment: "prod",
				Services:    []string{"   "},
			},
			want: "服务名为空或无效",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := validateConfig(&tc.config)
			if err == nil {
				t.Fatalf("validateConfig() expected error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateConfig() error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func TestValidateConfigAllowsAutoDiscoverWithoutServices(t *testing.T) {
	t.Helper()

	config := &Config{
		ProjectID:    "project-123",
		Environment:  "prod",
		AutoDiscover: true,
	}

	if err := validateConfig(config); err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
	if len(config.Services) != 0 {
		t.Fatalf("Services = %#v, want empty", config.Services)
	}
}

func TestValidateServiceName(t *testing.T) {
	t.Helper()

	valid := []string{"nginx", "my-api", "service_v2", "app.web"}
	for _, name := range valid {
		if err := validateServiceName(name); err != nil {
			t.Errorf("validateServiceName(%q) unexpected error: %v", name, err)
		}
	}

	invalid := []struct {
		name string
		want string
	}{
		{name: "", want: "为空或无效"},
		{name: ".", want: "为空或无效"},
		{name: " nginx", want: "首尾空白"},
		{name: "nginx ", want: "首尾空白"},
		{name: "my/service", want: "路径分隔符"},
		{name: `my\service`, want: "路径分隔符"},
		{name: "..", want: "路径穿越"},
		{name: "../../etc", want: "路径分隔符"},
		{name: "svc..name", want: "路径穿越"},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			err := validateServiceName(tc.name)
			if err == nil {
				t.Fatalf("validateServiceName(%q) expected error", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validateServiceName(%q) error = %q, want substring %q", tc.name, err.Error(), tc.want)
			}
		})
	}
}

func TestValidateServices(t *testing.T) {
	t.Helper()

	// 正常情况：trim 后校验通过
	services := []string{" nginx ", "api"}
	if err := validateServices(services); err != nil {
		t.Fatalf("validateServices() unexpected error: %v", err)
	}
	if services[0] != "nginx" || services[1] != "api" {
		t.Fatalf("validateServices() did not trim, got %#v", services)
	}

	// 异常情况：包含非法名称
	bad := []string{"nginx", "../evil"}
	if err := validateServices(bad); err == nil {
		t.Fatalf("validateServices() expected error for path traversal")
	}
}

func TestNormalizeRootFolder(t *testing.T) {
	t.Helper()

	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "   ", want: ""},
		{input: "/", want: ""},
		{input: "///team-a///", want: "/team-a"},
		{input: " /team-a/sub/ ", want: "/team-a/sub"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			if got := normalizeRootFolder(tc.input); got != tc.want {
				t.Fatalf("normalizeRootFolder(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
