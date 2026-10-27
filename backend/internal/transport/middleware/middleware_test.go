package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/org/metaesquemas/internal/config"
)

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })

func TestRealIPIgnoresSpoofedHeaderFromUntrustedPeer(t *testing.T) {
	var got netip.Addr
	h := RealIP([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = ClientIP(r.Context()) }))
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "203.0.113.9:1234"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got.String() != "203.0.113.9" {
		t.Fatalf("got %s", got)
	}
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "10.0.0.2:1"
	req.Header.Set("X-Forwarded-For", "6.6.6.6, 198.51.100.7, 10.0.0.5")
	h.ServeHTTP(httptest.NewRecorder(), req)
	if got.String() != "198.51.100.7" {
		t.Fatalf("got %s", got)
	}
}

func TestRateLimit429(t *testing.T) {
	h := RealIP(nil)(RateLimit(RateConfig{Name: "t", Anonymous: NewLimiter(1, 2)})(ok))
	codes := []int{}
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "203.0.113.1:1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	if codes[0] != 200 || codes[1] != 200 || codes[3] != 429 {
		t.Fatalf("codes=%v", codes)
	}
}

func TestGeoBlockByCountryHeaderAndCIDR(t *testing.T) {
	g, _ := NewGeoBlocker(config.Geo{Mode: "deny", Countries: map[string]struct{}{"RU": {}},
		DenyCIDRs: []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")}, CountryHeader: "CF-IPCountry"}, slog.Default())
	h := RealIP([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})(g.Middleware(ok))
	do := func(remote, xff, cc string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = remote
		if xff != "" {
			req.Header.Set("X-Forwarded-For", xff)
		}
		if cc != "" {
			req.Header.Set("CF-IPCountry", cc)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := do("10.0.0.2:1", "203.0.113.5", "RU"); c != 403 {
		t.Errorf("RU vía proxy de confianza: %d", c)
	}
	if c := do("10.0.0.2:1", "203.0.113.5", "CL"); c != 200 {
		t.Errorf("CL: %d", c)
	}
	if c := do("203.0.113.5:1", "", "RU"); c != 200 { // cabecera no confiable: se ignora
		t.Errorf("cabecera falsificada no debe bloquear/confiar: %d", c)
	}
	if c := do("198.51.100.20:1", "", ""); c != 403 {
		t.Errorf("CIDR denegado: %d", c)
	}
}
