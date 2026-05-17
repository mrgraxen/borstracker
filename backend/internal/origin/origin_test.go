package origin

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAllowRequest_trustProxy(t *testing.T) {
	c := NewChecker([]string{"http://localhost:8989"}, true)
	r := httptest.NewRequest("GET", "http://internal/api/v1/ws", nil)
	r.Header.Set("Origin", "https://bors.example.com")
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "bors.example.com")
	assert.True(t, c.AllowRequest(r))
}

func TestAllowRequest_rejectWrongOrigin(t *testing.T) {
	c := NewChecker([]string{"http://localhost:8989"}, false)
	r := httptest.NewRequest("GET", "http://internal/api/v1/ws", nil)
	r.Header.Set("Origin", "https://evil.com")
	assert.False(t, c.AllowRequest(r))
}

func TestAllowOrigin_explicitList(t *testing.T) {
	c := NewChecker([]string{"https://bors.example.com"}, false)
	assert.True(t, c.AllowOrigin("https://bors.example.com"))
	assert.False(t, c.AllowOrigin("https://other.com"))
}
