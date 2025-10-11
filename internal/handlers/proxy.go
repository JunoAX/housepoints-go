package handlers

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// ProxyToPythonBackend creates a reverse proxy handler that forwards requests to the Python backend
func ProxyToPythonBackend() gin.HandlerFunc {
	// Python backend service in K8s
	targetURL := "http://gamull-backend.production.svc.cluster.local:8000"
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Failed to parse Python backend URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Customize the proxy to handle errors and logging
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Log the proxied request
		log.Printf("🔀 Proxying to Python backend: %s %s", req.Method, req.URL.Path)
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("❌ Proxy error for %s %s: %v", r.Method, r.URL.Path, err)
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, `{"error": "Backend service unavailable"}`)
	}

	return func(c *gin.Context) {
		// Forward the request to Python backend
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
