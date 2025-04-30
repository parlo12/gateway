package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// newProxy builds a reverse-proxy to the given target URL.
func newProxy(target string) *httputil.ReverseProxy {
	u, err := url.Parse(target)
	if err != nil {
		log.Fatalf("invalid proxy target %q: %v", target, err)
	}
	p := httputil.NewSingleHostReverseProxy(u)

	// rewrite incoming URL before forwarding
	orig := p.Director
	p.Director = func(req *http.Request) {
		orig(req)
		req.Host = u.Host
	}
	return p
}

func main() {
	// build proxies
	authProxy := newProxy("http://auth-service:8082")
	contentProxy := newProxy("http://content-service:8083")

	// we’ll use Gin just for routing, but hand off to the stdlib proxy
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// AUTH: everything under /auth/* → auth-service
	r.Any("/auth/*proxyPath", gin.WrapH(authProxy))
	// also forward root signup/login so clients can do POST /signup
	r.Any("/signup", gin.WrapH(authProxy))
	r.Any("/login", gin.WrapH(authProxy))

	// CONTENT: everything under /content/* → content-service
	r.Any("/content/*proxyPath", gin.WrapH(contentProxy))

	// fallback 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"error": "not found"})
	})

	log.Println("Gateway listening on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("gateway error: %v", err)
	}
}
