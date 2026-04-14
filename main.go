// sub2api - A subscription converter API service
// Fork of Wei-Shaw/sub2api
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sub2api/sub2api/internal/api"
	"github.com/sub2api/sub2api/internal/config"
)

const (
	defaultPort    = 8080
	defaultHost    = "127.0.0.1" // changed from 0.0.0.0 - prefer localhost-only by default for personal use
	appName        = "sub2api"
	appVersion     = "dev"
)

func main() {
	// Parse command-line flags
	var (
		host       = flag.String("host", getEnvOrDefault("HOST", defaultHost), "Host to listen on")
		port       = flag.Int("port", getEnvOrDefaultInt("PORT", defaultPort), "Port to listen on")
		configFile = flag.String("config", getEnvOrDefault("CONFIG_FILE", ""), "Path to config file")
		version    = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize router
	router := api.NewRouter(cfg)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	log.Printf("Starting %s %s on %s", appName, appVersion, addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second, // tightened from 30s - 15s is plenty for local use
		WriteTimeout: 60 * time.Second, // increased write timeout to handle slow subscription fetches
		IdleTimeout:  90 * time.Second, // bumped idle timeout for keep-alive connections
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

// getEnvOrDefault returns the value of the environment variable or the default value.
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// getEnvOrDefaultInt returns the integer value of the environment variable or the default value.
func getEnvOrDefaultInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
