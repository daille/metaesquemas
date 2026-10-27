// Package middleware reúne las defensas perimetrales de la API.
package middleware

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type ctxKey int

const (
	keyClientIP ctxKey = iota
	keyViaTrusted
)

// ClientIP devuelve la IP real del cliente resuelta por RealIP.
func ClientIP(ctx context.Context) netip.Addr {
	ip, _ := ctx.Value(keyClientIP).(netip.Addr)
	return ip
}

// ViaTrustedProxy indica si la conexión TCP provino de un proxy de confianza.
func ViaTrustedProxy(ctx context.Context) bool {
	v, _ := ctx.Value(keyViaTrusted).(bool)
	return v
}

func inAny(ip netip.Addr, ps []netip.Prefix) bool {
	for _, p := range ps {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}

// RealIP resuelve la IP del cliente. X-Forwarded-For solo se interpreta si la
// conexión viene de un proxy de confianza; se recorre de derecha a izquierda
// descartando saltos de confianza, de modo que un cliente no pueda falsificarla.
func RealIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			peer, err := netip.ParseAddr(host)
			if err != nil {
				http.Error(w, "bad remote address", http.StatusBadRequest)
				return
			}
			peer = peer.Unmap()
			client, via := peer, false
			if inAny(peer, trusted) {
				via = true
				parts := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
				for i := len(parts) - 1; i >= 0; i-- {
					a, err := netip.ParseAddr(strings.TrimSpace(parts[i]))
					if err != nil {
						break
					}
					a = a.Unmap()
					client = a
					if !inAny(a, trusted) {
						break
					}
				}
			}
			ctx := context.WithValue(r.Context(), keyClientIP, client)
			ctx = context.WithValue(ctx, keyViaTrusted, via)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
