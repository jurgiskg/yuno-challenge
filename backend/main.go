package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"yuno-challenge/merchant"
	"yuno-challenge/sharedgin"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const defaultPort = "8080"

func main() {
	logger := newLogger()
	defer logger.Sync() //nolint:errcheck

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		if gin.Mode() == gin.ReleaseMode {
			logger.Fatal("API_KEY must be set when GIN_MODE=release")
		}
		logger.Warn("API_KEY is not set; /v1 endpoints are unauthenticated")
	}

	srv := &http.Server{Addr: ":" + port, Handler: newEngine(apiKey)}
	go func() {
		logger.Infof("Listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Failed to start gin engine: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Gin server forced to shutdown: %v", err)
	}
}

func newEngine(apiKey string) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	v1 := engine.Group("/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Health stays public for Render's health checks; everything else needs the key.
	authed := v1.Group("", sharedgin.RequireAPIKey(apiKey))

	merchantController := merchant.NewController()
	authed.POST("/authorizations", merchantController.CreateAuthorization)

	return engine
}

func newLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.Sugar()
}
