package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

// TestRepeatHandler is a focused suite for repeatHandler, buildRepeatGreeting, and the /repeat route
// as wired by newRouter + REPEAT.
func TestRepeatHandler(t *testing.T) {
	t.Run("buildRepeatGreeting is empty for zero lines", func(t *testing.T) {
		if got := buildRepeatGreeting(0); got != "" {
			t.Fatalf("buildRepeatGreeting(0) = %q, want empty", got)
		}
	})

	t.Run("buildRepeatGreeting treats negative as zero", func(t *testing.T) {
		if got := buildRepeatGreeting(-3); got != "" {
			t.Fatalf("buildRepeatGreeting(-3) = %q, want empty", got)
		}
	})

	t.Run("buildRepeatGreeting emits one line", func(t *testing.T) {
		got := buildRepeatGreeting(1)
		want := "Hello from Go!\n"
		if got != want {
			t.Fatalf("buildRepeatGreeting(1) = %q, want %q", got, want)
		}
	})

	t.Run("buildRepeatGreeting emits n newline-terminated lines", func(t *testing.T) {
		n := 4
		body := buildRepeatGreeting(n)
		if c := strings.Count(body, "Hello from Go!"); c != n {
			t.Fatalf("greeting line count: got %d, want %d, body: %q", c, n, body)
		}
		lines := strings.Count(body, "\n")
		if lines != n {
			t.Fatalf("newline count: got %d, want %d, body: %q", lines, n, body)
		}
	})

	t.Run("repeatHandler zero yields 200 and empty text body", func(t *testing.T) {
		eng := gin.New()
		eng.GET("/h", repeatHandler(0))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/h", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status: %d", w.Code)
		}
		if w.Body.String() != "" {
			t.Fatalf("body: %q, want empty", w.Body.String())
		}
		ct := w.Result().Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "text/") {
			t.Fatalf("Content-Type: %q, want text/*", ct)
		}
	})

	t.Run("repeatHandler two returns two greeting lines as plain text", func(t *testing.T) {
		eng := gin.New()
		eng.GET("/h", repeatHandler(2))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/h", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("status: %d", w.Code)
		}
		want := "Hello from Go!\n" + "Hello from Go!\n"
		if w.Body.String() != want {
			t.Fatalf("body = %q, want %q", w.Body.String(), want)
		}
	})

	t.Run("newRouter GET /repeat uses REPEAT env for line count", func(t *testing.T) {
		t.Setenv("REPEAT", "3")
		eng := newRouter()
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/repeat", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /repeat: %d, body: %q", w.Code, w.Body.String())
		}
		if strings.Count(w.Body.String(), "Hello from Go!") != 3 {
			t.Fatalf("body: %q, want 3 lines", w.Body.String())
		}
	})
}

// TestDbFunc exercises dbFunc with a mock database and the way newRouterForDB registers /db.
func TestDbFunc(t *testing.T) {
	execSQL := func(s string) string { return regexp.QuoteMeta(s) }
	wantTick := time.Date(2020, 4, 1, 12, 0, 0, 0, time.UTC)
	succeededRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"tick"}).AddRow(wantTick)
	}

	t.Run("200 with Read from DB on happy path", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })

		mock.ExpectExec(execSQL(sqlCreateTicks)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(execSQL(sqlInsertTick)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(execSQL(sqlSelectTicks)).WillReturnRows(succeededRows())

		eng := gin.New()
		eng.GET("/db", dbFunc(db))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/db", nil))

		if w.Code != http.StatusOK {
			t.Fatalf("status: %d body: %s", w.Code, w.Body.String())
		}
		if !strings.HasPrefix(w.Body.String(), "Read from DB: ") {
			t.Fatalf("body: %q", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "2020-") {
			t.Fatalf("body should include formatted time, got: %q", w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})

	t.Run("500 when create table fails", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })

		mock.ExpectExec(execSQL(sqlCreateTicks)).WillReturnError(sql.ErrConnDone)

		eng := gin.New()
		eng.GET("/db", dbFunc(db))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/db", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status: %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Error creating database table") {
			t.Fatalf("body: %q", w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})

	t.Run("500 when insert fails", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })

		mock.ExpectExec(execSQL(sqlCreateTicks)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(execSQL(sqlInsertTick)).WillReturnError(sql.ErrTxDone)

		eng := gin.New()
		eng.GET("/db", dbFunc(db))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/db", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status: %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Error incrementing tick") {
			t.Fatalf("body: %q", w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})

	t.Run("500 when select fails", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })

		mock.ExpectExec(execSQL(sqlCreateTicks)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(execSQL(sqlInsertTick)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(execSQL(sqlSelectTicks)).WillReturnError(sql.ErrNoRows)

		eng := gin.New()
		eng.GET("/db", dbFunc(db))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/db", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status: %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Error reading ticks") {
			t.Fatalf("body: %q", w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})

	t.Run("500 when row scan fails", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("sqlmock: %v", err)
		}
		t.Cleanup(func() { _ = db.Close() })

		rows := sqlmock.NewRows([]string{"tick"}).AddRow("not-a-timestamp")
		mock.ExpectExec(execSQL(sqlCreateTicks)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec(execSQL(sqlInsertTick)).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectQuery(execSQL(sqlSelectTicks)).WillReturnRows(rows)

		eng := gin.New()
		eng.GET("/db", dbFunc(db))
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/db", nil))
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("status: %d: %q", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Error scanning ticks") {
			t.Fatalf("body: %q", w.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations: %v", err)
		}
	})

	t.Run("newRouterForDB with nil has no /db route", func(t *testing.T) {
		eng := newRouterForDB(nil)
		w := httptest.NewRecorder()
		eng.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/db", nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d, body: %q", w.Code, w.Body.String())
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
