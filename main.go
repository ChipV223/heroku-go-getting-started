package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	_ "github.com/lib/pq"
	"github.com/russross/blackfriday"
)

// SQL for the /db tick demo (Postgres). Kept as constants so tests and dbFunc stay aligned.
const (
	sqlCreateTicks = `CREATE TABLE IF NOT EXISTS ticks (tick timestamp)`
	sqlInsertTick  = `INSERT INTO ticks VALUES (now())`
	sqlSelectTicks = `SELECT tick FROM ticks`
)

// buildRepeatGreeting returns n lines of the fixed greeting (each line ends with '\n'). Negative n is treated as 0.
func buildRepeatGreeting(n int) string {
	if n < 0 {
		n = 0
	}
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		b.WriteString("Hello from Go!\n")
	}
	return b.String()
}

func repeatCountFromEnv() int {
	s := os.Getenv("REPEAT")
	if s == "" {
		return 5
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Printf("Error converting $REPEAT to an int (%q: %v) – using default 5\n", s, err)
		return 5
	}
	return n
}

// dbFunc returns a handler that creates the ticks table, inserts a row, and prints stored timestamps.
// It is intended for use with a Postgres *sql.DB; tests may pass a *sql.DB backed by sqlmock.
func dbFunc(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := db.Exec(sqlCreateTicks); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error creating database table: %q", err))
			return
		}

		if _, err := db.Exec(sqlInsertTick); err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error incrementing tick: %q", err))
			return
		}

		rows, err := db.Query(sqlSelectTicks)
		if err != nil {
			c.String(http.StatusInternalServerError,
				fmt.Sprintf("Error reading ticks: %q", err))
			return
		}

		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var tick time.Time
			if err := rows.Scan(&tick); err != nil {
				c.String(http.StatusInternalServerError,
					fmt.Sprintf("Error scanning ticks: %q", err))
				return
			}
			c.String(http.StatusOK, fmt.Sprintf("Read from DB: %s\n", tick.String()))
		}
	}
}

// repeatHandler returns a Gin handler that writes buildRepeatGreeting(r) with status 200.
func repeatHandler(r int) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.String(http.StatusOK, buildRepeatGreeting(r))
	}
}

func getRequiredPort() (string, error) {
	port := os.Getenv("PORT")
	if port == "" {
		return "", errors.New("$PORT must be set")
	}
	return port, nil
}

// openAppDB opens Postgres when DATABASE_URL is set. An empty or whitespace-only URL yields (nil, nil) so
// the app can still serve static routes in tests and local dev without a database.
func openAppDB() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, nil
	}
	return sql.Open("postgres", dsn)
}

// newRouterForDB builds the application router. If db is non-nil, GET /db is registered. Callers
// in tests may use a sqlmock or real *sql.DB.
func newRouterForDB(db *sql.DB) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", handleIndex)
	router.GET("/mark", handleMark)
	router.GET("/repeat", repeatHandler(repeatCountFromEnv()))
	if db != nil {
		router.GET("/db", dbFunc(db))
	}
	return router
}

// newRouter wires openAppDB and newRouterForDB. It exits the process on sql.Open error when DATABASE_URL is set.
func newRouter() *gin.Engine {
	db, err := openAppDB()
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}
	return newRouterForDB(db)
}

func handleIndex(c *gin.Context) {
	c.HTML(http.StatusOK, "index.tmpl.html", nil)
}

func handleMark(c *gin.Context) {
	c.String(http.StatusOK, string(blackfriday.Run([]byte("**hi!**"))))
}

func runApp() error {
	port, err := getRequiredPort()
	if err != nil {
		return err
	}
	router := newRouter()
	if err := router.Run(":" + port); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := runApp(); err != nil {
		log.Fatal(err)
	}
}
