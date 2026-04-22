package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestMain configures the Gin engine for the rest of the tests.
func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestGetRequiredPort(t *testing.T) {
	t.Run("returns error when PORT is not set", func(t *testing.T) {
		t.Setenv("PORT", "")
		_, err := getRequiredPort()
		if err == nil {
			t.Fatal("getRequiredPort: expected error when PORT is empty")
		}
		if !strings.Contains(err.Error(), "PORT") {
			t.Fatalf("getRequiredPort: error should mention PORT: %v", err)
		}
	})

	t.Run("returns port when env is set", func(t *testing.T) {
		t.Setenv("PORT", "5432")
		port, err := getRequiredPort()
		if err != nil {
			t.Fatalf("getRequiredPort: %v", err)
		}
		if port != "5432" {
			t.Fatalf("port: got %q, want 5432", port)
		}
	})
}

func TestNewRouter(t *testing.T) {
	t.Run("serves home route", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET /: want status %d, got %d, body: %q", http.StatusOK, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Getting Started with Go on Heroku") {
			t.Fatalf("GET /: body should include app title, got: %q", w.Body.String())
		}
	})

	t.Run("serves /mark for markdown", func(t *testing.T) {
		r := newRouter()
		req := httptest.NewRequest(http.MethodGet, "/mark", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET /mark: want status %d, got %d, body: %q", http.StatusOK, w.Code, w.Body.String())
		}
	})
}

func TestHandleIndex(t *testing.T) {
	t.Run("responds with HTML 200 and index content", func(t *testing.T) {
		r := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /: %d, body: %q", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "jumbotron") {
			t.Fatalf("expected jumbotron in index HTML, got: %q", w.Body.String())
		}
	})

	t.Run("includes the go-getting-started source link", func(t *testing.T) {
		r := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /: %d, body: %q", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "github.com/heroku/go-getting-started") {
			t.Fatalf("expected GitHub link, got: %q", w.Body.String())
		}
	})
}

func TestHandleMark(t *testing.T) {
	t.Run("renders markdown with strong emphasis for hi", func(t *testing.T) {
		r := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/mark", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /mark: %d, body: %q", w.Code, w.Body.String())
		}
		body := w.Body.String()
		if !strings.Contains(body, "hi!") {
			t.Fatalf("expected 'hi!' in body: %q", body)
		}
		if !strings.Contains(body, "<strong>") || !strings.Contains(body, "</strong>") {
			t.Fatalf("expected <strong> in markdown output: %q", body)
		}
	})

	t.Run("string response is non-empty and uses a text content type", func(t *testing.T) {
		r := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/mark", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /mark: %d", w.Code)
		}
		if w.Body.Len() == 0 {
			t.Fatal("empty body for /mark")
		}
		ct := w.Result().Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "text/") {
			t.Fatalf("expected text/* content type for c.String, got %q", ct)
		}
	})
}

func TestRunApp(t *testing.T) {
	t.Run("fails when PORT is not set", func(t *testing.T) {
		t.Setenv("PORT", "")
		err := runApp()
		if err == nil {
			t.Fatal("runApp: expected error when PORT is empty")
		}
		if !strings.Contains(err.Error(), "PORT") {
			t.Fatalf("runApp: %v", err)
		}
	})

	t.Run("fails when address port is invalid and Listen fails", func(t *testing.T) {
		// 70000 is out of range for a TCP port; http.Server should return an error
		// before the server can accept (runApp's Run is non-blocking on that failure).
		t.Setenv("PORT", "70000")
		err := runApp()
		if err == nil {
			t.Fatal("runApp: expected error for invalid port 70000")
		}
	})
}

// TestEntrypointMain wires the same router and templates as the main process. The
// top-level main() only calls runApp with log.Fatal; those paths are covered by
// TestRunApp and the HTTP checks below.
func TestEntrypointMain(t *testing.T) {
	t.Run("index page includes the logo section like production", func(t *testing.T) {
		r := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /: %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "lang-logo") {
			t.Fatalf("expected lang-logo in page: %q", w.Body.String())
		}
	})

	t.Run("mark route matches newRouter registration used by runApp", func(t *testing.T) {
		r := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/mark", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /mark: %d", w.Code)
		}
		if w.Body.String() == "" {
			t.Fatal("empty /mark body")
		}
	})
}
