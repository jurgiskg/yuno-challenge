package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"yuno-challenge/acquirer"
	"yuno-challenge/acquirer/acq1"
	"yuno-challenge/acquirer/acq2"
	"yuno-challenge/acquirer/acq3"
	"yuno-challenge/authorization"
	"yuno-challenge/merchant"
	"yuno-challenge/sharedgin"
	"yuno-challenge/testdata"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	defaultPort = "8080"
	// defaultAcquirerOrder is the routing order used when ACQUIRER_ORDER is unset.
	defaultAcquirerOrder = "AcquirerOne,AcquirerTwo,AcquirerThree"
)

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

	acquirers, err := routedAcquirers(os.Getenv("ACQUIRER_ORDER"))
	if err != nil {
		logger.Fatalf("Invalid ACQUIRER_ORDER: %v", err)
	}
	processor, err := acquirer.NewProcessor(acquirers, authorization.NewStore(), logger)
	if err != nil {
		logger.Fatalf("Failed to create processor: %v", err)
	}

	srv := &http.Server{Addr: ":" + port, Handler: newEngine(apiKey, processor)}
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

func newEngine(apiKey string, processor merchant.Processor) *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	v1 := engine.Group("/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Health stays public for Render's health checks; everything else needs the key.
	authed := v1.Group("", sharedgin.RequireAPIKey(apiKey))

	merchantController := merchant.NewController(processor)
	authed.POST("/authorizations", merchantController.CreateAuthorization)

	return engine
}

// routedAcquirers returns the mock acquirers named in order, a comma-separated
// list such as "AcquirerTwo,AcquirerOne". A subset is allowed, e.g. a single
// acquirer to measure the no-failover baseline. Empty uses defaultAcquirerOrder.
func routedAcquirers(order string) ([]acquirer.Acquirer, error) {
	if strings.TrimSpace(order) == "" {
		order = defaultAcquirerOrder
	}
	one, err := acq1.New(testdata.AcquirerOneRules)
	if err != nil {
		return nil, err
	}
	two, err := acq2.New(testdata.AcquirerTwoRules)
	if err != nil {
		return nil, err
	}
	three, err := acq3.New(testdata.AcquirerThreeRules)
	if err != nil {
		return nil, err
	}
	available := map[string]acquirer.Acquirer{one.Name(): one, two.Name(): two, three.Name(): three}

	var routed []acquirer.Acquirer
	for _, name := range strings.Split(order, ",") {
		a, ok := available[strings.TrimSpace(name)]
		if !ok {
			return nil, fmt.Errorf("unknown acquirer %q, must be one of AcquirerOne, AcquirerTwo, AcquirerThree", name)
		}
		routed = append(routed, a)
	}
	return routed, nil
}

func newLogger() *zap.SugaredLogger {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	return logger.Sugar()
}
