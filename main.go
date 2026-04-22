package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/heroku/x/hmetrics/onload"
	"github.com/russross/blackfriday"
)

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
