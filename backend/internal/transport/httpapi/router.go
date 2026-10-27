package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/org/metaesquemas/internal/application"
	"github.com/org/metaesquemas/internal/config"
	"github.com/org/metaesquemas/internal/transport/middleware"
)

// NewRouter arma la cadena de defensa. Orden (de afuera hacia adentro):
// RealIP → Recoverer → GeoBlock → SecurityHeaders → CORS → Timeout → RateLimit → handler.
func NewRouter(cfg *config.Config, svc *application.Service, geo *middleware.GeoBlocker, log *slog.Logger) http.Handler {
	h := &Handlers{svc: svc, log: log, adminToken: cfg.AdminToken, maxUpload: cfg.MaxUploadBytes}

	valid := middleware.KeySet(cfg.APIKeys)
	general := middleware.RateLimit(middleware.RateConfig{Name: "api", Anonymous: middleware.NewLimiter(cfg.RateAnon.RPS, cfg.RateAnon.Burst),
		Keyed: middleware.NewLimiter(cfg.RateKey.RPS, cfg.RateKey.Burst), ValidKey: valid})
	admin := middleware.RateLimit(middleware.RateConfig{Name: "admin", Anonymous: middleware.NewLimiter(cfg.RateAdmin.RPS, cfg.RateAdmin.Burst)})
	adminRead := middleware.RateLimit(middleware.RateConfig{Name: "admin-read", Anonymous: middleware.NewLimiter(cfg.RateAdminRead.RPS, cfg.RateAdminRead.Burst)})

	r := chi.NewRouter()
	r.Use(middleware.RealIP(cfg.TrustedProxies))
	r.Use(chimw.Recoverer)
	if geo != nil {
		r.Use(geo.Middleware)
	}
	r.Use(middleware.SecurityHeaders(cfg.HSTS))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(chimw.Timeout(cfg.RequestTimeout))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	r.Route("/api/v1", func(api chi.Router) {
		api.Group(func(read chi.Router) {
			read.Use(general)
			read.Use(chimw.Compress(5, "application/json", "application/xml", "text/plain"))
			read.Get("/stats", h.stats)
			read.Get("/facets", h.facets)
			read.Get("/namespaces", h.namespaces)
			read.Get("/schemas", h.list)
			read.Get("/schemas/{id}", h.get)
			read.Get("/schemas/{id}/artifacts/{kind}", h.artifact)
			read.Get("/schemas/{id}/dependencies", h.tree)
			read.Get("/schemas/{id}/graph", h.graph)
		})
		// El límite por IP se aplica ANTES de validar el token (frena adivinación de credenciales).
		api.Group(func(a chi.Router) {
			a.Use(admin)
			a.Use(h.adminAuth)
			a.Post("/admin/schemas", h.upload)
		})
		api.Group(func(a chi.Router) {
			a.Use(adminRead)
			a.Use(h.adminAuth)
			a.Get("/admin/jobs/{id}", h.job)
		})
	})
	return r
}
