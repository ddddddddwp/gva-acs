package initialize

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/gin-gonic/gin"
)

func TestSetupEngineUsesRuntimeSettingsAfterConstruction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRuntime := config.CurrentRuntime()
	previousLegacy := tr069Global.GlobalConfig
	t.Cleanup(func() {
		config.StoreRuntime(previousRuntime.Settings)
		tr069Global.GlobalConfig = previousLegacy
	})

	config.StoreRuntime(config.TR069Config{})
	tr069Global.GlobalConfig = &config.TR069Config{}
	engine := SetupEngine()

	firstDir := t.TempDir()
	config.StoreRuntime(config.TR069Config{
		InfoLogEnable:    true,
		InfoLogDir:       firstDir,
		DumpMaxBytes:     5,
		DumpRedactAuth:   true,
		DumpRedactCookie: true,
	})
	first := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader("first-body-is-long"))
	first.Header.Set("Authorization", "Bearer first-secret")
	first.Header.Set("Cookie", "session=first-cookie")
	engine.ServeHTTP(httptest.NewRecorder(), first)

	firstLog := readRuntimeInfoLog(t, firstDir)
	if !strings.Contains(firstLog, "Authorization: <redacted>") {
		t.Fatalf("first request authorization was not redacted:\n%s", firstLog)
	}
	if !strings.Contains(firstLog, "Cookie: <redacted>") {
		t.Fatalf("first request cookie was not redacted:\n%s", firstLog)
	}
	if strings.Contains(firstLog, "first-secret") || strings.Contains(firstLog, "first-cookie") {
		t.Fatalf("first request leaked a redacted value:\n%s", firstLog)
	}
	if !strings.Contains(firstLog, "first\n\n<TRUNCATED>") {
		t.Fatalf("first request did not use the five-byte runtime limit:\n%s", firstLog)
	}

	secondDir := t.TempDir()
	config.StoreRuntime(config.TR069Config{
		InfoLogEnable:    true,
		InfoLogDir:       secondDir,
		DumpMaxBytes:     1024,
		DumpRedactAuth:   false,
		DumpRedactCookie: false,
	})
	second := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader("second-body-is-complete"))
	second.Header.Set("Authorization", "Bearer second-secret")
	second.Header.Set("Cookie", "session=second-cookie")
	engine.ServeHTTP(httptest.NewRecorder(), second)

	secondLog := readRuntimeInfoLog(t, secondDir)
	for _, want := range []string{"Bearer second-secret", "session=second-cookie", "second-body-is-complete"} {
		if !strings.Contains(secondLog, want) {
			t.Fatalf("second request log missing %q:\n%s", want, secondLog)
		}
	}
	if strings.Contains(secondLog, "<TRUNCATED>") {
		t.Fatalf("second request retained the earlier byte limit:\n%s", secondLog)
	}

	beforeDisable := secondLog
	config.StoreRuntime(config.TR069Config{})
	disabled := httptest.NewRequest(http.MethodPost, "/acs", strings.NewReader("must-not-be-logged"))
	engine.ServeHTTP(httptest.NewRecorder(), disabled)
	if afterDisable := readRuntimeInfoLog(t, secondDir); afterDisable != beforeDisable {
		t.Fatalf("disabled runtime appended to the existing log:\n%s", afterDisable)
	}
}

func TestFileIngressRouteDoesNotEnterRawDumpMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRuntime := config.CurrentRuntime()
	t.Cleanup(func() { config.StoreRuntime(previousRuntime.Settings) })
	logDir := t.TempDir()
	config.StoreRuntime(config.TR069Config{InfoLogEnable: true, InfoLogDir: logDir, DumpMaxBytes: 4096})
	engine := setupEngine(map[string]gin.HandlerFunc{
		"/acs/log": func(c *gin.Context) {
			_, _ = io.Copy(io.Discard, c.Request.Body)
			c.Status(http.StatusNoContent)
		},
	})
	request := httptest.NewRequest(http.MethodPut, "/acs/log", strings.NewReader("binary-log-payload-must-not-enter-raw-dump"))
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d", response.Code)
	}
	path := filepath.Join(logDir, time.Now().Format("2006-01-02"), "tr069info.log")
	contents, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read raw log: %v", err)
	}
	if strings.Contains(string(contents), "binary-log-payload-must-not-enter-raw-dump") {
		t.Fatalf("file ingress payload leaked into raw dump:\n%s", contents)
	}
}

func readRuntimeInfoLog(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, time.Now().Format("2006-01-02"), "tr069info.log")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read runtime info log: %v", err)
	}
	return string(contents)
}
