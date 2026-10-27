package middleware

import (
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"github.com/oschwald/geoip2-golang"

	"github.com/org/metaesquemas/internal/config"
)

// GeoBlocker filtra por país (GeoLite2/GeoIP2) y por bloques CIDR.
// Orden de evaluación: IP privada → bloques CIDR denegados → país.
type GeoBlocker struct {
	cfg config.Geo
	db  *geoip2.Reader
	log *slog.Logger
}

func NewGeoBlocker(cfg config.Geo, log *slog.Logger) (*GeoBlocker, error) {
	g := &GeoBlocker{cfg: cfg, log: log}
	if cfg.DBPath != "" {
		db, err := geoip2.Open(cfg.DBPath)
		switch {
		case err == nil:
			g.db = db
		case cfg.FailClosed:
			return nil, err
		default:
			log.Warn("no se pudo abrir la base GeoIP; el geobloqueo por país queda inactivo", "path", cfg.DBPath, "err", err)
		}
	} else {
		log.Warn("GEOIP_DB no configurado: el geobloqueo por país depende de GEO_COUNTRY_HEADER")
	}
	return g, nil
}

func (g *GeoBlocker) Close() {
	if g.db != nil {
		_ = g.db.Close()
	}
}

func (g *GeoBlocker) country(r *http.Request, ip netip.Addr) (string, bool) {
	// Un proxy/CDN de confianza (p.ej. Cloudflare) puede entregar el país ya resuelto.
	if g.cfg.CountryHeader != "" && ViaTrustedProxy(r.Context()) {
		if cc := strings.ToUpper(strings.TrimSpace(r.Header.Get(g.cfg.CountryHeader))); len(cc) == 2 && cc != "XX" && cc != "T1" {
			return cc, true
		}
	}
	if g.db == nil {
		return "", false
	}
	rec, err := g.db.Country(net.IP(ip.AsSlice()))
	if err != nil || rec.Country.IsoCode == "" {
		return "", false
	}
	return rec.Country.IsoCode, true
}

func (g *GeoBlocker) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r.Context())
		if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			next.ServeHTTP(w, r)
			return
		}
		if inAny(ip, g.cfg.DenyCIDRs) {
			g.deny(w, ip, "cidr", "")
			return
		}
		cc, ok := g.country(r, ip)
		switch {
		case !ok && g.cfg.FailClosed:
			g.deny(w, ip, "unresolved", "")
			return
		case !ok:
			next.ServeHTTP(w, r)
			return
		}
		_, listed := g.cfg.Countries[cc]
		if (g.cfg.Mode == "deny" && listed) || (g.cfg.Mode == "allow" && !listed) {
			g.deny(w, ip, "country", cc)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (g *GeoBlocker) deny(w http.ResponseWriter, ip netip.Addr, reason, cc string) {
	g.log.Warn("geoblock", "ip", ip.String(), "reason", reason, "country", cc)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"error":{"code":"region_blocked","message":"Acceso no disponible desde su ubicación."}}`))
}
