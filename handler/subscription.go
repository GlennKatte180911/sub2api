// Package handler provides HTTP handlers for the sub2api service.
// It handles subscription URL fetching and format conversion.
package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default HTTP client timeout for fetching subscriptions
	DefaultTimeout = 15 * time.Second
	// MaxResponseSize limits the response body to 10MB
	MaxResponseSize = 10 * 1024 * 1024
)

// SubscriptionHandler handles incoming requests to fetch and convert
// subscription URLs into various proxy client formats.
type SubscriptionHandler struct {
	client     *http.Client
	backendURL string
}

// NewSubscriptionHandler creates a new SubscriptionHandler with the given
// backend conversion URL and HTTP timeout.
func NewSubscriptionHandler(backendURL string, timeout time.Duration) *SubscriptionHandler {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &SubscriptionHandler{
		client: &http.Client{
			Timeout: timeout,
		},
		backendURL: strings.TrimRight(backendURL, "/"),
	}
}

// ServeHTTP handles the HTTP request, extracts the target subscription URL
// and desired output format from query parameters, fetches the subscription,
// and proxies the converted response back to the client.
func (h *SubscriptionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	subURL := q.Get("url")
	if subURL == "" {
		http.Error(w, "missing required parameter: url", http.StatusBadRequest)
		return
	}

	// Validate that the provided URL is well-formed
	if _, err := url.ParseRequestURI(subURL); err != nil {
		http.Error(w, fmt.Sprintf("invalid url parameter: %v", err), http.StatusBadRequest)
		return
	}

	// Build the upstream backend request URL, forwarding all query params
	upstreamURL := fmt.Sprintf("%s/sub?%s", h.backendURL, r.URL.RawQuery)

	upstreamReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, upstreamURL, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to build upstream request: %v", err), http.StatusInternalServerError)
		return
	}

	// Forward relevant headers from the original request
	forwardHeaders(r, upstreamReq)

	resp, err := h.client.Do(upstreamReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("upstream request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy upstream response headers to our response
	for key, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(key, v)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Stream the response body with a size limit
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize)
	if _, err := io.Copy(w, limitedReader); err != nil {
		// Response has already started; we can only log the error
		fmt.Printf("error streaming response body: %v\n", err)
	}
}

// forwardHeaders copies a safe subset of request headers to the upstream request.
func forwardHeaders(src *http.Request, dst *http.Request) {
	allowed := []string{
		"User-Agent",
		"Accept",
		"Accept-Language",
		"Accept-Encoding",
	}
	for _, h := range allowed {
		if v := src.Header.Get(h); v != "" {
			dst.Header.Set(h, v)
		}
	}
}
