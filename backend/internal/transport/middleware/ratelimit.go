package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	lim  *rate.Limiter
	seen time.Time
}

// Limiter es un token bucket por clave con expiración de entradas.
type Limiter struct {
	mu       sync.Mutex
	m        map[string]*visitor
	rps      rate.Limit
	burst    int
	max      int
	overflow *rate.Limiter // cubo compartido si el mapa se llena (ataque con IPs masivas)
}

func NewLimiter(rps float64, burst int) *Limiter {
	l := &Limiter{
		m: make(map[string]*visitor), rps: rate.Limit(rps), burst: burst, max: 200_000,
		overflow: rate.NewLimiter(rate.Limit(rps*50), burst*50),
	}
	go func() {
		for range time.Tick(time.Minute) {
			l.mu.Lock()
			for k, v := range l.m {
				if time.Since(v.seen) > 5*time.Minute {
					delete(l.m, k)
				}
			}
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *Limiter) allow(key string) (bool, float64) {
	l.mu.Lock()
	v, ok := l.m[key]
	if !ok {
		if len(l.m) >= l.max {
			l.mu.Unlock()
			return l.overflow.Allow(), 0
		}
		v = &visitor{lim: rate.NewLimiter(l.rps, l.burst)}
		l.m[key] = v
	}
	v.seen = time.Now()
	l.mu.Unlock()
	return v.lim.Allow(), v.lim.Tokens()
}

// ipKey agrupa IPv6 por /64 para que un atacante no eluda el límite rotando direcciones.
func ipKey(ip netip.Addr) string {
	if ip.Is6() {
		p, _ := ip.Prefix(64)
		return p.String()
	}
	return ip.String()
}

type RateConfig struct {
	Name      string
	Anonymous *Limiter
	Keyed     *Limiter              // opcional: límite mayor para clientes con API key
	ValidKey  func(key string) bool // opcional
}

// RateLimit aplica límites por IP o por API key (cabecera X-API-Key).
func RateLimit(cfg RateConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lim, key := cfg.Anonymous, cfg.Name+"|ip|"+ipKey(ClientIP(r.Context()))
			if k := r.Header.Get("X-API-Key"); k != "" && cfg.Keyed != nil && cfg.ValidKey != nil && cfg.ValidKey(k) {
				sum := sha256.Sum256([]byte(k))
				lim, key = cfg.Keyed, cfg.Name+"|key|"+hex.EncodeToString(sum[:8])
			}
			ok, tokens := lim.allow(key)
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(lim.burst))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(int(math.Max(0, tokens))))
			if !ok {
				retry := int(math.Ceil(1 / float64(lim.rps)))
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"code":"rate_limited","message":"Demasiadas solicitudes. Reintente más tarde."}}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// KeySet valida API keys en tiempo constante.
func KeySet(keys map[string]struct{}) func(string) bool {
	return func(k string) bool {
		found := 0
		for have := range keys {
			found |= subtle.ConstantTimeCompare([]byte(have), []byte(k))
		}
		return found == 1
	}
}

// MaxInFlight limita las solicitudes concurrentes (load shedding): protege a Oxigraph
// de consultas costosas simultáneas devolviendo 503 en lugar de encolar sin límite.
func MaxInFlight(n int) func(http.Handler) http.Handler {
	sem := make(chan struct{}, n)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
				next.ServeHTTP(w, r)
			default:
				w.Header().Set("Retry-After", "2")
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"error":{"code":"busy","message":"Servicio ocupado. Reintente en unos segundos."}}`))
			}
		})
	}
}
