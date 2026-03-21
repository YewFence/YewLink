package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputMode(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "default", value: "", want: outputModeRelative},
		{name: "absolute uppercase", value: "ABSOLUTE", want: outputModeAbsolute},
		{name: "relative lowercase", value: "relative", want: outputModeRelative},
		{name: "invalid fallback", value: "weird", want: outputModeRelative},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OUTPUT_MODE", tc.value)
			if got := resolveOutputMode(); got != tc.want {
				t.Fatalf("resolveOutputMode() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveAutoPrune(t *testing.T) {
	t.Helper()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "default enabled", value: "", want: true},
		{name: "explicit true", value: "true", want: true},
		{name: "mixed case true", value: "TRUE", want: true},
		{name: "explicit false", value: "false", want: false},
		{name: "invalid treated as false", value: "nope", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AUTO_PRUNE", tc.value)
			if got := resolveAutoPrune(); got != tc.want {
				t.Fatalf("resolveAutoPrune() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestResolveProjectDir(t *testing.T) {
	t.Helper()

	t.Run("flag has priority", func(t *testing.T) {
		flagValue := filepath.Join("projects", "demo-app")
		t.Setenv("PROJECT_DIR", filepath.Join("env-projects", "ignored"))

		name, absPath := resolveProjectDir(flagValue)
		if name != "demo-app" {
			t.Fatalf("name = %q, want %q", name, "demo-app")
		}
		if absPath != flagValue {
			t.Fatalf("absPath = %q, want %q", absPath, flagValue)
		}
	})

	t.Run("env fallback", func(t *testing.T) {
		envValue := filepath.Join("env-projects", "demo-app")
		t.Setenv("PROJECT_DIR", envValue)

		name, absPath := resolveProjectDir("")
		if name != "demo-app" {
			t.Fatalf("name = %q, want %q", name, "demo-app")
		}
		if absPath != envValue {
			t.Fatalf("absPath = %q, want %q", absPath, envValue)
		}
	})
}

func TestExtractDirName(t *testing.T) {
	t.Helper()

	path := filepath.Join("root", "demo-app")
	if got := extractDirName(path); got != "demo-app" {
		t.Fatalf("extractDirName(%q) = %q, want %q", path, got, "demo-app")
	}
}

func TestBuildLnCommand(t *testing.T) {
	t.Helper()

	t.Run("relative mode", func(t *testing.T) {
		cfg := GeneratorConfig{
			projectName: "infisical-agent",
			projectPath: filepath.Join("root", "infisical-agent"),
			outputMode:  outputModeRelative,
		}

		want := "cd ../nginx && ln -sf " + filepath.Join("..", "infisical-agent", "secrets", "nginx.env") + " .env"
		if got := buildLnCommand("nginx", cfg); got != want {
			t.Fatalf("buildLnCommand() = %q, want %q", got, want)
		}
	})

	t.Run("absolute mode", func(t *testing.T) {
		cfg := GeneratorConfig{
			projectName: "infisical-agent",
			projectPath: filepath.Join("root", "infisical-agent"),
			outputMode:  outputModeAbsolute,
		}

		want := "ln -sf " + filepath.Join("root", "infisical-agent", "secrets", "nginx.env") + " " + filepath.Join("root", "infisical-agent", "nginx", ".env")
		if got := buildLnCommand("nginx", cfg); got != want {
			t.Fatalf("buildLnCommand() = %q, want %q", got, want)
		}
	})
}

func TestBuildMvCommand(t *testing.T) {
	t.Helper()

	t.Run("relative mode", func(t *testing.T) {
		cfg := GeneratorConfig{outputMode: outputModeRelative}
		want := "cd ../nginx && mv .env .env.bak"
		if got := buildMvCommand("nginx", cfg); got != want {
			t.Fatalf("buildMvCommand() = %q, want %q", got, want)
		}
	})

	t.Run("absolute mode", func(t *testing.T) {
		cfg := GeneratorConfig{
			projectPath: filepath.Join("root", "infisical-agent"),
			outputMode:  outputModeAbsolute,
		}

		target := filepath.Join("root", "infisical-agent", "nginx", ".env")
		want := "mv " + target + " " + target + ".bak"
		if got := buildMvCommand("nginx", cfg); got != want {
			t.Fatalf("buildMvCommand() = %q, want %q", got, want)
		}
	})
}

func TestBuildSecretPath(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		root    string
		service string
		want    string
	}{
		{name: "empty root", root: "", service: "nginx", want: "/nginx"},
		{name: "service with slashes", root: "/team-a", service: "/nginx/", want: "/team-a/nginx"},
		{name: "empty service returns root", root: "/team-a", service: "   ", want: "/team-a"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := buildSecretPath(tc.root, tc.service); got != tc.want {
				t.Fatalf("buildSecretPath(%q, %q) = %q, want %q", tc.root, tc.service, got, tc.want)
			}
		})
	}
}

func TestGetWorkingDirName(t *testing.T) {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	want := filepath.Base(cwd)
	if want == "" || want == "." {
		want = "infisical-agent"
	}

	if got := getWorkingDirName(); got != want {
		t.Fatalf("getWorkingDirName() = %q, want %q", got, want)
	}
}

func TestCleanStaleSecrets(t *testing.T) {
	t.Helper()

	t.Run("removes stale env files only", func(t *testing.T) {
		dir := t.TempDir()
		files := []string{"nginx.env", "api.env", "old.env", "README.txt"}
		for _, name := range files {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, []byte(name), 0o600); err != nil {
				t.Fatalf("WriteFile(%q) error = %v", name, err)
			}
		}
		if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
			t.Fatalf("Mkdir() error = %v", err)
		}

		if err := cleanStaleSecrets(dir, []string{"nginx", "api"}); err != nil {
			t.Fatalf("cleanStaleSecrets() error = %v", err)
		}

		for _, name := range []string{"nginx.env", "api.env", "README.txt"} {
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Fatalf("Stat(%q) error = %v", name, err)
			}
		}

		if _, err := os.Stat(filepath.Join(dir, "old.env")); !os.IsNotExist(err) {
			t.Fatalf("old.env should be removed, stat error = %v", err)
		}
		if info, err := os.Stat(filepath.Join(dir, "nested")); err != nil || !info.IsDir() {
			t.Fatalf("nested directory should remain, stat error = %v", err)
		}
	})

	t.Run("missing directory is ignored", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		if err := cleanStaleSecrets(dir, []string{"nginx"}); err != nil {
			t.Fatalf("cleanStaleSecrets() error = %v", err)
		}
	})
}
