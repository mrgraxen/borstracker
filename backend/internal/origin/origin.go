package origin

import (
	"net/http"
	"strings"
)

// Checker validates browser Origin for CORS and WebSocket.
type Checker struct {
	allowed    map[string]struct{}
	trustProxy bool
}

func NewChecker(allowedOrigins []string, trustProxy bool) *Checker {
	m := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" {
			m[o] = struct{}{}
		}
	}
	return &Checker{allowed: m, trustProxy: trustProxy}
}

func (c *Checker) AllowOrigin(origin string) bool {
	if origin == "" {
		return true
	}
	if _, ok := c.allowed[origin]; ok {
		return true
	}
	return false
}

func (c *Checker) AllowRequest(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if c.AllowOrigin(origin) {
		return true
	}
	if !c.trustProxy || origin == "" {
		return false
	}
	return origin == InferFromProxy(r)
}

// InferFromProxy rebuilds the public site origin from reverse-proxy headers.
func InferFromProxy(r *http.Request) string {
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		if r.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return proto + "://" + host
}

func ParseList(primary string, extra string) []string {
	var out []string
	seen := make(map[string]struct{})
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(primary)
	for _, part := range strings.Split(extra, ",") {
		add(part)
	}
	return out
}
