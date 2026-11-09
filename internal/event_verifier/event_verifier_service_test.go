package event_verifier

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"net/http/httptest"
)

func TestRandomTokenIsOpaqueAndHashIsStable(t *testing.T) {
	first, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	second, err := randomToken()
	if err != nil {
		t.Fatal(err)
	}
	if first == second || len(first) < 40 {
		t.Fatalf("token tidak cukup acak: first=%q second=%q", first, second)
	}
	if tokenHash(first) == first || len(tokenHash(first)) != 64 {
		t.Fatalf("hash token tidak valid: %q", tokenHash(first))
	}
	if tokenHash(first) != tokenHash(first) {
		t.Fatal("hash token tidak deterministik")
	}
}

func TestBearerOnlyAcceptsBearerScheme(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		header string
		want   string
	}{
		{"Bearer secret", "secret"},
		{"bearer secret", "secret"},
		{"Basic secret", ""},
		{"secret", ""},
		{"Bearer   ", ""},
	}
	for _, test := range tests {
		t.Run(strings.ReplaceAll(test.header, " ", "_"), func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/", nil)
			ctx.Request.Header.Set("Authorization", test.header)
			if got := bearer(ctx); got != test.want {
				t.Fatalf("bearer() = %q, want %q", got, test.want)
			}
		})
	}
}
