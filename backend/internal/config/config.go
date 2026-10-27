// Package config carga la configuración desde variables de entorno (12-factor).
package config

import (
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

type Rate struct {
	RPS   float64
	Burst int
}

type Geo struct {
	DBPath        string
	Mode          string // "deny" (lista negra) | "allow" (lista blanca)
	Countries     map[string]struct{}
	DenyCIDRs     []netip.Prefix
	FailClosed    bool
	CountryHeader string
}

type Config struct {
	Addr           string
	StoreDir       string // repositorio de esquemas pre-calculados (solo lectura)
	InboxDir       string // cola de trabajos para el worker (escritura)
	JobsDir        string // estado de trabajos (lectura)
	AdminToken     string
	APIKeys        map[string]struct{}
	AllowedOrigins []string
	TrustedProxies []netip.Prefix
	HSTS           bool

	RateAnon      Rate
	RateKey       Rate
	RateAdmin     Rate // cargas de archivos (estricto)
	RateAdminRead Rate // consulta del estado de trabajos (sondeo frecuente)
	Geo           Geo

	RequestTimeout time.Duration
	MaxUploadBytes int64
	ReloadEvery    time.Duration
}

func Load() (*Config, error) {
	c := &Config{
		Addr:       env("ADDR", ":8080"),
		StoreDir:   env("STORE_DIR", "/data/store"),
		InboxDir:   env("INBOX_DIR", "/data/inbox"),
		JobsDir:    env("JOBS_DIR", "/data/jobs"),
		AdminToken: os.Getenv("ADMIN_TOKEN"),
		HSTS:       envBool("HSTS", true),
		APIKeys:    map[string]struct{}{},

		RateAnon:      Rate{envFloat("RATE_ANON_RPS", 20), envInt("RATE_ANON_BURST", 60)},
		RateKey:       Rate{envFloat("RATE_KEY_RPS", 100), envInt("RATE_KEY_BURST", 300)},
		RateAdmin:     Rate{envFloat("RATE_ADMIN_RPS", 0.5), envInt("RATE_ADMIN_BURST", 5)},
		RateAdminRead: Rate{envFloat("RATE_ADMIN_READ_RPS", 5), envInt("RATE_ADMIN_READ_BURST", 20)},

		RequestTimeout: envDuration("REQUEST_TIMEOUT", 15*time.Second),
		MaxUploadBytes: int64(envInt("MAX_UPLOAD_BYTES", 256<<10)),
		ReloadEvery:    envDuration("RELOAD_EVERY", 2*time.Second),
	}
	for _, k := range split(os.Getenv("API_KEYS")) {
		c.APIKeys[k] = struct{}{}
	}
	c.AllowedOrigins = split(env("CORS_ALLOWED_ORIGINS", "http://localhost:8080"))
	var err error
	if c.TrustedProxies, err = prefixes(env("TRUSTED_PROXIES", "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16")); err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	c.Geo = Geo{DBPath: os.Getenv("GEOIP_DB"), Mode: strings.ToLower(env("GEO_MODE", "deny")), Countries: map[string]struct{}{},
		FailClosed: envBool("GEO_FAIL_CLOSED", false), CountryHeader: os.Getenv("GEO_COUNTRY_HEADER")}
	for _, cc := range split(env("GEO_COUNTRIES", "RU,KP")) {
		c.Geo.Countries[strings.ToUpper(cc)] = struct{}{}
	}
	if c.Geo.DenyCIDRs, err = prefixes(os.Getenv("GEO_DENY_CIDRS")); err != nil {
		return nil, fmt.Errorf("GEO_DENY_CIDRS: %w", err)
	}
	if c.Geo.Mode != "deny" && c.Geo.Mode != "allow" {
		return nil, fmt.Errorf("GEO_MODE debe ser deny o allow")
	}
	return c, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil {
		return v
	}
	return def
}
func envFloat(k string, def float64) float64 {
	if v, err := strconv.ParseFloat(os.Getenv(k), 64); err == nil {
		return v
	}
	return def
}
func envBool(k string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(k)); err == nil {
		return v
	}
	return def
}
func envDuration(k string, def time.Duration) time.Duration {
	if v, err := time.ParseDuration(os.Getenv(k)); err == nil {
		return v
	}
	return def
}
func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
func prefixes(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, p := range split(s) {
		pf, err := netip.ParsePrefix(p)
		if err != nil {
			return nil, err
		}
		out = append(out, pf.Masked())
	}
	return out, nil
}
