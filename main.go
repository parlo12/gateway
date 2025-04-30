package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	// run Gin in release by default
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// config via env (can override in docker-compose)
	gatewayPort := getEnv("GATEWAY_PORT", "8080")
	authSvcURL := getEnv("AUTH_SERVICE_URL", "http://auth-service:8082")
	contentSvcURL := getEnv("CONTENT_SERVICE_URL", "http://content-service:8083")

	// health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "up"})
	})

	// build reverse-proxies
	authProxy := mustNewProxy(authSvcURL)
	contentProxy := mustNewProxy(contentSvcURL)

	// proxy /signup and /login to auth-service
	router.Any("/signup", wrapProxy(authProxy))
	router.Any("/login", wrapProxy(authProxy))
	// also proxy any /auth/... paths (to catch /auth/register, etc.)
	router.Any("/auth/*proxyPath", wrapProxy(authProxy))

	// proxy content endpoints
	router.Any("/content/*proxyPath", wrapProxy(contentProxy))

	log.Printf("▶ Gateway listening on :%s, forwarding auth→%s, content→%s",
		gatewayPort, authSvcURL, contentSvcURL)

	if err := router.Run(":" + gatewayPort); err != nil {
		log.Fatalf("gateway failed: %v", err)
	}
}

// mustNewProxy parses targetURL and returns a ReverseProxy or panics.
func mustNewProxy(targetURL string) *httputil.ReverseProxy {
	u, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("bad proxy URL %q: %v", targetURL, err)
	}
	return httputil.NewSingleHostReverseProxy(u)
}

// wrapProxy returns a Gin handler which delegates to the given proxy.
func wrapProxy(p *httputil.ReverseProxy) gin.HandlerFunc {
	return func(c *gin.Context) {
		// you could rewrite path here if you need to strip prefixes
		p.ServeHTTP(c.Writer, c.Request)
	}
}

// getEnv reads an env var or returns the default.
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
