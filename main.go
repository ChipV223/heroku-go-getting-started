package main

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/russross/blackfriday"
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

func newRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger())
	router.LoadHTMLGlob("templates/*.tmpl.html")
	router.Static("/static", "static")

	router.GET("/", handleIndex)
	router.GET("/mark", handleMark)
	router.GET("/repeat", repeatHandler(repeatCountFromEnv()))
	return router
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
