package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadCredentialFile(t *testing.T) {
	t.Helper()

	t.Run("trimmed content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "client-id")
		if err := os.WriteFile(path, []byte("  secret-value \n"), 0o600); err != nil {
			t.Fatalf("write credential: %v", err)
		}

		got, err := readCredentialFile(path)
		if err != nil {
			t.Fatalf("readCredentialFile() error = %v", err)
		}
		if got != "secret-value" {
			t.Fatalf("readCredentialFile() = %q, want %q", got, "secret-value")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := readCredentialFile(filepath.Join(t.TempDir(), "missing")); err == nil {
			t.Fatalf("readCredentialFile() expected error")
		}
	})
}

func TestFetchToken(t *testing.T) {
	t.Helper()

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			if r.URL.Path != "/api/v1/auth/universal-auth/login" {
				t.Fatalf("path = %s, want /api/v1/auth/universal-auth/login", r.URL.Path)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}

			var payload map[string]string
			if err := json.Unmarshal(body, &payload); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if payload["clientId"] != "client-id" {
				t.Fatalf("clientId = %q, want %q", payload["clientId"], "client-id")
			}
			if payload["clientSecret"] != "client-secret" {
				t.Fatalf("clientSecret = %q, want %q", payload["clientSecret"], "client-secret")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"accessToken":"token-123"}`))
		}))
		defer server.Close()

		token, err := fetchToken(server.URL, "client-id", "client-secret")
		if err != nil {
			t.Fatalf("fetchToken() error = %v", err)
		}
		if token != "token-123" {
			t.Fatalf("fetchToken() = %q, want %q", token, "token-123")
		}
	})

	t.Run("non 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad credentials", http.StatusUnauthorized)
		}))
		defer server.Close()

		_, err := fetchToken(server.URL, "client-id", "client-secret")
		if err == nil {
			t.Fatalf("fetchToken() expected error")
		}
		if !strings.Contains(err.Error(), "HTTP 401") {
			t.Fatalf("fetchToken() error = %q, want HTTP status", err.Error())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken":`))
		}))
		defer server.Close()

		if _, err := fetchToken(server.URL, "client-id", "client-secret"); err == nil {
			t.Fatalf("fetchToken() expected error")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"accessToken":""}`))
		}))
		defer server.Close()

		_, err := fetchToken(server.URL, "client-id", "client-secret")
		if err == nil {
			t.Fatalf("fetchToken() expected error")
		}
		if !strings.Contains(err.Error(), "accessToken 为空") {
			t.Fatalf("fetchToken() error = %q, want empty token message", err.Error())
		}
	})
}

func TestDiscoverFolders(t *testing.T) {
	t.Helper()

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/api/v2/folders" {
				t.Fatalf("path = %s, want /api/v2/folders", r.URL.Path)
			}
			if got := r.URL.Query().Get("projectId"); got != "project-123" {
				t.Fatalf("projectId = %q, want %q", got, "project-123")
			}
			if got := r.URL.Query().Get("environment"); got != "prod" {
				t.Fatalf("environment = %q, want %q", got, "prod")
			}
			if got := r.URL.Query().Get("path"); got != "/team-a" {
				t.Fatalf("path = %q, want %q", got, "/team-a")
			}
			if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
				t.Fatalf("Authorization = %q, want %q", got, "Bearer token-123")
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"folders":[{"name":"nginx"},{"name":""},{"name":"api"}]}`))
		}))
		defer server.Close()

		folders, err := discoverFolders(server.URL, "project-123", "prod", "/team-a", "token-123")
		if err != nil {
			t.Fatalf("discoverFolders() error = %v", err)
		}

		want := []string{"nginx", "api"}
		if len(folders) != len(want) || folders[0] != want[0] || folders[1] != want[1] {
			t.Fatalf("discoverFolders() = %#v, want %#v", folders, want)
		}
	})

	t.Run("non 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "forbidden", http.StatusForbidden)
		}))
		defer server.Close()

		_, err := discoverFolders(server.URL, "project-123", "prod", "/", "token-123")
		if err == nil {
			t.Fatalf("discoverFolders() expected error")
		}
		if !strings.Contains(err.Error(), "HTTP 403") {
			t.Fatalf("discoverFolders() error = %q, want HTTP status", err.Error())
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"folders":`))
		}))
		defer server.Close()

		if _, err := discoverFolders(server.URL, "project-123", "prod", "/", "token-123"); err == nil {
			t.Fatalf("discoverFolders() expected error")
		}
	})
}
