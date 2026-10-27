package httpapi

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/org/metaesquemas/internal/application"
	"github.com/org/metaesquemas/internal/domain"
)

type Handlers struct {
	svc        *application.Service
	log        *slog.Logger
	adminToken string
	maxUpload  int64
}

func intq(r *http.Request, k string) int { n, _ := strconv.Atoi(r.URL.Query().Get(k)); return n }

func (h *Handlers) stats(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=10")
	writeJSON(w, 200, h.svc.Catalog.Stats())
}
func (h *Handlers) facets(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=10")
	writeJSON(w, 200, h.svc.Catalog.Facets())
}
func (h *Handlers) namespaces(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=10")
	writeJSON(w, 200, map[string]any{"items": h.svc.Catalog.Namespaces()})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var tipos []string
	if t := q.Get("tipo"); t != "" {
		tipos = strings.Split(t, ",")
	}
	res, err := h.svc.List(domainParams(q.Get("q"), q.Get("namespace"), q.Get("responsable"), q.Get("vigencia"), q.Get("sort"), q.Get("usada"), tipos, intq(r, "limit"), intq(r, "offset")))
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	writeJSON(w, 200, res)
}

func domainParams(q, ns, resp, vig, sort, usada string, tipos []string, limit, offset int) domain.ListParams {
	return domain.ListParams{Q: q, Namespace: ns, Responsable: resp, Vigencia: vig, Sort: sort, Usada: usada, Tipos: tipos, Limit: limit, Offset: offset}
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.Get(chi.URLParam(r, "id"), intq(r, "version"))
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=10")
	writeJSON(w, 200, d)
}

// artifact sirve una proyección ya generada (lectura directa de disco, sin compilar).
func (h *Handlers) artifact(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Artifact(chi.URLParam(r, "id"), chi.URLParam(r, "kind"), intq(r, "version"))
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	defer a.Close()
	hd := w.Header()
	hd.Set("Content-Type", a.ContentType)
	hd.Set("ETag", a.ETag)
	if a.Immutable {
		hd.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		hd.Set("Cache-Control", "public, max-age=30, must-revalidate")
	}
	if r.URL.Query().Get("download") == "1" {
		hd.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, a.FileName))
	}
	http.ServeContent(w, r, a.FileName, a.ModTime, a.Reader)
}

func (h *Handlers) tree(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.Tree(chi.URLParam(r, "id"), r.URL.Query().Get("direction"), intq(r, "depth"))
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	writeJSON(w, 200, t)
}

func (h *Handlers) graph(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.Graph(chi.URLParam(r, "id"), intq(r, "depth"))
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	writeJSON(w, 200, g)
}

// ---- Administración (ingesta) ----

func (h *Handlers) adminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if h.adminToken == "" || subtle.ConstantTimeCompare([]byte(tok), []byte(h.adminToken)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="admin"`)
			writeJSON(w, 401, map[string]any{"error": map[string]string{"code": "unauthorized", "message": "Token inválido."}})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// upload recibe un .tsp (multipart "file" o cuerpo crudo con ?fileName=) y lo encola. Responde 202:
// la validación estricta, la compilación y la publicación las hace el worker.
func (h *Handlers) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUpload+(64<<10))
	var name string
	var src []byte
	var err error
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		var f multipart.File
		var hdr *multipart.FileHeader
		if err = r.ParseMultipartForm(1 << 20); err == nil {
			f, hdr, err = r.FormFile("file")
		}
		if err != nil {
			writeErr(w, h.log, fmt.Errorf("%w: falta el campo 'file' o supera el tamaño máximo", domain.ErrInvalid))
			return
		}
		defer f.Close()
		name = path.Base(strings.ReplaceAll(hdr.Filename, "\\", "/"))
		src, err = io.ReadAll(io.LimitReader(f, h.maxUpload+1))
	} else {
		name = r.URL.Query().Get("fileName")
		src, err = io.ReadAll(io.LimitReader(r.Body, h.maxUpload+1))
	}
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			writeErr(w, h.log, fmt.Errorf("%w: archivo demasiado grande", domain.ErrInvalid))
			return
		}
		writeErr(w, h.log, err)
		return
	}
	id, err := h.svc.Upload(r.Context(), name, r.Header.Get("X-Uploader"), src)
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	h.log.Info("archivo encolado", "job", id, "file", name, "bytes", len(src))
	writeJSON(w, http.StatusAccepted, map[string]any{"jobId": id, "status": "queued", "statusUrl": "/api/v1/admin/jobs/" + id})
}

func (h *Handlers) job(w http.ResponseWriter, r *http.Request) {
	b, err := h.svc.Ingest.Job(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, h.log, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(b)
}
