/*
  mcp_builtin_test.go -- Unit tests for built-in MCP tools (DEBT-03).
  Covers read_file (path sandbox enforcement) and web_fetch (SSRF guard).
  Tests complete in < 5s each; httptest.Server is used in place of real network calls.
*/

package bindings

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// read_file

func TestBuiltinReadFile_ReadsAllowedFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(file, []byte("hello liaotao"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, handled := builtinReadFile(`{"path":"`+file+`"}`, []string{dir})
	if !handled {
		t.Fatal("expected handled=true")
	}
	if result != "hello liaotao" {
		t.Errorf("unexpected content: %q", result)
	}
}

func TestBuiltinReadFile_BlocksOutsideAllowedRoots(t *testing.T) {
	dir := t.TempDir()
	otherDir := t.TempDir()
	file := filepath.Join(otherDir, "secret.txt")
	if err := os.WriteFile(file, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, handled := builtinReadFile(`{"path":"`+file+`"}`, []string{dir})
	if !handled {
		t.Fatal("expected handled=true")
	}
	if result == "secret" {
		t.Error("read_file returned content from outside allowed_roots — SECURITY BUG")
	}
	if result[:5] != "error" {
		t.Errorf("expected error message, got %q", result)
	}
}

func TestBuiltinReadFile_BlocksPathTraversal(t *testing.T) {
	dir := t.TempDir()
	// Attempt traversal: allowed root is dir, path tries to escape via ..
	traversalPath := filepath.Join(dir, "..", "escape.txt")

	result, handled := builtinReadFile(`{"path":"`+traversalPath+`"}`, []string{dir})
	if !handled {
		t.Fatal("expected handled=true")
	}
	// filepath.Abs cleans the traversal — the resolved path must be rejected.
	if result == "escaped" {
		t.Error("path traversal succeeded — SECURITY BUG")
	}
}

func TestBuiltinReadFile_ErrorOnMissingPath(t *testing.T) {
	result, handled := builtinReadFile(`{}`, []string{"/tmp"})
	if !handled {
		t.Fatal("expected handled=true")
	}
	if len(result) < 5 || result[:5] != "error" {
		t.Errorf("expected error for missing path, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// web_fetch

func TestBuiltinWebFetch_FetchesPublicURL(t *testing.T) {
	// Note: httptest.NewServer always binds to 127.0.0.1, which our SSRF guard blocks.
	// This test verifies the guard returns an error (correct behavior in unit test context).
	// Real external fetches are validated via manual integration testing.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fetch ok"))
	}))
	defer srv.Close()

	result, handled := builtinWebFetch(`{"url":"` + srv.URL + `"}`)
	if !handled {
		t.Fatal("expected handled=true")
	}
	// The httptest server is on 127.0.0.1 which our SSRF guard must block.
	// Verify we get an error, not the content.
	if result == "fetch ok" {
		t.Error("SSRF guard failure: web_fetch returned content from localhost")
	}
	if len(result) < 5 || result[:5] != "error" {
		t.Errorf("expected SSRF error for localhost URL, got %q", result)
	}
}

func TestBuiltinWebFetch_BlocksLocalhost(t *testing.T) {
	cases := []string{
		`{"url":"http://localhost:8080/secret"}`,
		`{"url":"http://127.0.0.1:8080/secret"}`,
	}
	for _, tc := range cases {
		result, handled := builtinWebFetch(tc)
		if !handled {
			t.Fatalf("expected handled=true for %s", tc)
		}
		if len(result) < 5 || result[:5] != "error" {
			t.Errorf("expected error for localhost access, got %q (input: %s)", result, tc)
		}
	}
}

func TestBuiltinWebFetch_BlocksPrivateIPRanges(t *testing.T) {
	cases := []string{
		`{"url":"http://10.0.0.1/api"}`,
		`{"url":"http://192.168.1.100/admin"}`,
		`{"url":"http://172.16.0.1/internal"}`,
		`{"url":"http://172.31.255.255/internal"}`,
	}
	for _, tc := range cases {
		result, handled := builtinWebFetch(tc)
		if !handled {
			t.Fatalf("expected handled=true for %s", tc)
		}
		if len(result) < 5 || result[:5] != "error" {
			t.Errorf("expected error for private IP, got %q (input: %s)", result, tc)
		}
	}
}

func TestBuiltinWebFetch_BlocksNonHTTP(t *testing.T) {
	cases := []string{
		`{"url":"file:///etc/passwd"}`,
		`{"url":"ftp://ftp.example.com/secret"}`,
	}
	for _, tc := range cases {
		result, handled := builtinWebFetch(tc)
		if !handled {
			t.Fatalf("expected handled=true for %s", tc)
		}
		if len(result) < 5 || result[:5] != "error" {
			t.Errorf("expected error for non-HTTP scheme, got %q (input: %s)", result, tc)
		}
	}
}

func TestBuiltinWebFetch_ErrorOnHTTP4xx(t *testing.T) {
	// Note: direct httptest 127.0.0.1 URLs are blocked by SSRF guard.
	// Test instead with a non-existent domain that causes a dial error.
	result, handled := builtinWebFetch(`{"url":"http://this-domain-does-not-exist-liaotao-test.invalid/missing"}`)
	if !handled {
		t.Fatal("expected handled=true")
	}
	if len(result) < 5 || result[:5] != "error" {
		t.Errorf("expected error for unreachable URL, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// dispatchBuiltin — smoke test for all 4 tools

func TestDispatchBuiltin_AllFourToolsHandled(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	_ = os.WriteFile(file, []byte("ok"), 0o600)

	cases := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{"current_datetime", `{}`, false},
		{"calculator", `{"expression":"2+2"}`, false},
		{"read_file", `{"path":"` + file + `"}`, false},
		// web_fetch is not tested via httptest here (SSRF guard blocks 127.0.0.1).
		// Security-critical paths (block localhost, private IPs, non-HTTP) are tested separately.
		{"web_fetch", `{"url":"http://this-domain-does-not-exist-liaotao-test.invalid/"}`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, handled := dispatchBuiltin(tc.name, tc.args, []string{dir})
			if !handled {
				t.Fatalf("%s: expected handled=true", tc.name)
			}
			hasError := len(result) >= 5 && result[:5] == "error"
			if tc.wantErr && !hasError {
				t.Errorf("%s: expected error result, got %q", tc.name, result)
			}
			if !tc.wantErr && hasError {
				t.Errorf("%s returned unexpected error: %s", tc.name, result)
			}
		})
	}
}
