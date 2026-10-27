package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/org/metaesquemas/internal/adapters/store"
	"github.com/org/metaesquemas/internal/application"
	"github.com/org/metaesquemas/internal/config"
	"github.com/org/metaesquemas/internal/testutil"
	"github.com/org/metaesquemas/internal/transport/middleware"
)

type env struct {
	h    http.Handler
	root string
}

func setup(t *testing.T, mut func(*config.Config)) env {
	dir := testutil.NewStore(t)
	data := t.TempDir()
	cfg := &config.Config{
		StoreDir: dir, InboxDir: filepath.Join(data, "inbox"), JobsDir: filepath.Join(data, "jobs"), AdminToken: "s3cr3to-largo-de-prueba",
		APIKeys: map[string]struct{}{"clave-integrador": {}}, AllowedOrigins: []string{"https://catalogo.example"}, HSTS: true,
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")},
		RateAnon:       config.Rate{RPS: 1000, Burst: 1000}, RateKey: config.Rate{RPS: 1000, Burst: 1000}, RateAdmin: config.Rate{RPS: 1000, Burst: 1000}, RateAdminRead: config.Rate{RPS: 1000, Burst: 1000},
		Geo:            config.Geo{Mode: "deny", Countries: map[string]struct{}{"RU": {}}, CountryHeader: "CF-IPCountry"},
		RequestTimeout: 5 * time.Second, MaxUploadBytes: 1024, ReloadEvery: time.Hour,
	}
	if mut != nil {
		mut(cfg)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	cat, err := store.New(cfg.StoreDir, cfg.ReloadEvery, log)
	if err != nil {
		t.Fatal(err)
	}
	svc := &application.Service{Catalog: cat, Ingest: store.NewQueue(cfg.InboxDir, cfg.JobsDir), MaxUpload: cfg.MaxUploadBytes}
	geo, _ := middleware.NewGeoBlocker(cfg.Geo, log)
	return env{h: NewRouter(cfg, svc, geo, log), root: data}
}

func (e env) do(method, url string, body io.Reader, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, body)
	req.RemoteAddr = "203.0.113.7:4000"
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	e.h.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("JSON inválido (%d): %s", rec.Code, rec.Body.String())
	}
}

func TestLecturaDelCatalogo(t *testing.T) {
	e := setup(t, nil)
	var list struct {
		Total int
		Items []struct {
			ID, VigenciaEstado string
			UsedByCount        int
		}
	}
	rec := e.do("GET", "/api/v1/schemas?tipo=basico&usada=si", nil, nil)
	decode(t, rec, &list)
	if rec.Code != 200 || list.Total != 1 || list.Items[0].ID != "Pub.Base.Rut" || list.Items[0].UsedByCount != 3 || list.Items[0].VigenciaEstado != "vigente" {
		t.Fatalf("lista: %d %+v", rec.Code, list)
	}
	var st struct {
		Total  int
		ByTipo map[string]int
	}
	decode(t, e.do("GET", "/api/v1/stats", nil, nil), &st)
	if st.Total != 7 || st.ByTipo["documental"] != 1 {
		t.Errorf("stats: %+v", st)
	}
	var det struct {
		ID, Tipo  string
		Available map[string]string
		Source    struct{ Text string }
	}
	rec = e.do("GET", "/api/v1/schemas/Pub.Comun.Persona", nil, nil)
	decode(t, rec, &det)
	if rec.Code != 200 || det.Tipo != "estructurado" || det.Available["jsonschema"] == "" || det.Source.Text == "" {
		t.Fatalf("detalle: %d %s", rec.Code, rec.Body.String())
	}
	var tr struct{ Children []struct{ ID string } }
	decode(t, e.do("GET", "/api/v1/schemas/Pub.Base.Rut/dependencies?direction=usedBy&depth=1", nil, nil), &tr)
	if len(tr.Children) != 3 {
		t.Errorf("usado por: %+v", tr)
	}
	var g struct{ Nodes, Edges []any }
	decode(t, e.do("GET", "/api/v1/schemas/Pub.Comun.Persona/graph", nil, nil), &g)
	if len(g.Nodes) < 4 || len(g.Edges) < 3 {
		t.Errorf("grafo: %+v", g)
	}
	for url, want := range map[string]int{
		"/api/v1/schemas/No.Existe": 404, "/api/v1/schemas/..%2F..%2Fetc%2Fpasswd": 400, "/api/v1/schemas?tipo=hack": 400, "/api/v1/schemas?vigencia=x": 400,
		"/api/v1/schemas?limit=abc": 200, "/api/v1/schemas/Pub.Base.Rut/dependencies?direction=zzz": 400, "/api/v1/schemas/Pub.Base.Rut/artifacts/exe": 400,
		"/api/v1/schemas/Pub.Base.Rut/artifacts/jsonschema?version=1": 200, "/api/v1/schemas/Pub.Base.Rut/artifacts/xsd?version=1": 404, "/healthz": 204,
	} {
		if rec := e.do("GET", url, nil, nil); rec.Code != want {
			t.Errorf("%s -> %d, esperaba %d: %s", url, rec.Code, want, rec.Body.String())
		}
	}
}

func TestArtefactosCacheYDescarga(t *testing.T) {
	e := setup(t, nil)
	rec := e.do("GET", "/api/v1/schemas/Pub.Comun.Persona/artifacts/jsonschema", nil, nil)
	if rec.Code != 200 || rec.Body.String() != `{"type":"object"}` || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("jsonschema: %d %q %s", rec.Code, rec.Body.String(), rec.Header().Get("Content-Type"))
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "must-revalidate") {
		t.Errorf("la versión vigente debe revalidarse: %s", cc)
	}
	etag := rec.Header().Get("ETag")
	if rec2 := e.do("GET", "/api/v1/schemas/Pub.Comun.Persona/artifacts/jsonschema", nil, map[string]string{"If-None-Match": etag}); rec2.Code != http.StatusNotModified {
		t.Errorf("If-None-Match debe dar 304, dio %d", rec2.Code)
	}
	if cc := e.do("GET", "/api/v1/schemas/Pub.Comun.Persona/artifacts/jsonschema?version=1", nil, nil).Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("versión explícita debe ser inmutable: %s", cc)
	}
	rec = e.do("GET", "/api/v1/schemas/Pub.Comun.Persona/artifacts/tsp?download=1", nil, nil)
	if cd := rec.Header().Get("Content-Disposition"); !strings.Contains(cd, `filename="Pub.Comun.Persona.tsp"`) {
		t.Errorf("descarga: %q", cd)
	}
	if rec := e.do("GET", "/api/v1/schemas/Pub.Base.Estado/artifacts/proto?version=2", nil, nil); rec.Code != 404 {
		t.Errorf("ruta manipulada en meta.json debe dar 404, dio %d", rec.Code)
	}
}

func multipartBody(t *testing.T, field, name, content string) (*bytes.Buffer, string) {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, _ := mw.CreateFormFile(field, name)
	fw.Write([]byte(content))
	mw.Close()
	return &b, mw.FormDataContentType()
}

func TestCargaAdministrativa(t *testing.T) {
	e := setup(t, nil)
	tok := map[string]string{"Authorization": "Bearer s3cr3to-largo-de-prueba", "X-Uploader": "ana@gob.example"}
	src := "namespace A.B;\nmodel M {}"

	// sin token / token incorrecto
	body, ct := multipartBody(t, "file", "m.tsp", src)
	if rec := e.do("POST", "/api/v1/admin/schemas", body, map[string]string{"Content-Type": ct}); rec.Code != 401 {
		t.Errorf("sin token: %d", rec.Code)
	}
	body, ct = multipartBody(t, "file", "m.tsp", src)
	if rec := e.do("POST", "/api/v1/admin/schemas", body, map[string]string{"Content-Type": ct, "Authorization": "Bearer otro"}); rec.Code != 401 {
		t.Errorf("token incorrecto: %d", rec.Code)
	}
	// carga válida: 202 y queda en la cola, sin compilar
	body, ct = multipartBody(t, "file", "m.tsp", src)
	h := map[string]string{"Content-Type": ct}
	for k, v := range tok {
		h[k] = v
	}
	rec := e.do("POST", "/api/v1/admin/schemas", body, h)
	var out struct{ JobID, Status string }
	decode(t, rec, &out)
	if rec.Code != 202 || out.Status != "queued" {
		t.Fatalf("carga: %d %s", rec.Code, rec.Body.String())
	}
	if b, _ := os.ReadFile(filepath.Join(e.root, "inbox", out.JobID, "source.tsp")); string(b) != src {
		t.Error("el archivo no quedó en la cola del worker")
	}
	if rec := e.do("GET", "/api/v1/admin/jobs/"+out.JobID, nil, tok); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"queued"`) {
		t.Errorf("estado del trabajo: %d %s", rec.Code, rec.Body.String())
	}
	if rec := e.do("GET", "/api/v1/admin/jobs/"+out.JobID, nil, nil); rec.Code != 401 {
		t.Errorf("el estado requiere token: %d", rec.Code)
	}
	if rec := e.do("GET", "/api/v1/admin/jobs/..%2F..%2Fx", nil, tok); rec.Code != 400 && rec.Code != 404 {
		t.Errorf("id de trabajo inválido: %d", rec.Code)
	}
	// cuerpo crudo con ?fileName=
	if rec := e.do("POST", "/api/v1/admin/schemas?fileName=crudo.tsp", strings.NewReader(src), tok); rec.Code != 202 {
		t.Errorf("cuerpo crudo: %d %s", rec.Code, rec.Body.String())
	}
	// rechazos
	for name, c := range map[string]struct{ file, content string }{
		"extensión": {"m.js", src}, "vacío": {"v.tsp", ""}, "binario": {"b.tsp", "a\x00b"}, "grande": {"g.tsp", strings.Repeat("x", 2000)}, "utf8": {"u.tsp", "\xff\xfe"},
	} {
		body, ct := multipartBody(t, "file", c.file, c.content)
		hh := map[string]string{"Content-Type": ct}
		for k, v := range tok {
			hh[k] = v
		}
		if rec := e.do("POST", "/api/v1/admin/schemas", body, hh); rec.Code != 400 {
			t.Errorf("%s debe rechazarse con 400, dio %d: %s", name, rec.Code, rec.Body.String())
		}
	}
	// multipart: el nombre se reduce a su base (no hay recorrido de rutas); cuerpo crudo: la ruta se rechaza
	body, ct = multipartBody(t, "file", "../../etc/m.tsp", src)
	hh := map[string]string{"Content-Type": ct}
	for k, v := range tok {
		hh[k] = v
	}
	rec = e.do("POST", "/api/v1/admin/schemas", body, hh)
	decode(t, rec, &out)
	if jb, _ := os.ReadFile(filepath.Join(e.root, "inbox", out.JobID, "job.json")); rec.Code != 202 || !strings.Contains(string(jb), `"fileName":"m.tsp"`) {
		t.Errorf("el nombre debe quedar saneado: %d %s", rec.Code, jb)
	}
	if rec := e.do("POST", "/api/v1/admin/schemas?fileName=../m.tsp", strings.NewReader(src), tok); rec.Code != 400 {
		t.Errorf("fileName con ruta debe rechazarse: %d", rec.Code)
	}
	if entries, _ := os.ReadDir(filepath.Join(e.root, "inbox")); len(entries) != 3 {
		t.Errorf("solo las 3 cargas válidas deben estar en la cola: %d", len(entries))
	}
}

func TestSeguridadPerimetral(t *testing.T) {
	e := setup(t, nil)
	rec := e.do("GET", "/api/v1/stats", nil, map[string]string{"Origin": "https://catalogo.example"})
	for h, want := range map[string]string{"X-Frame-Options": "DENY", "X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer", "Access-Control-Allow-Origin": "https://catalogo.example"} {
		if got := rec.Header().Get(h); got != want {
			t.Errorf("%s=%q, esperaba %q", h, got, want)
		}
	}
	if !strings.Contains(rec.Header().Get("Strict-Transport-Security"), "max-age=") || !strings.Contains(rec.Header().Get("Content-Security-Policy"), "default-src 'none'") {
		t.Error("faltan HSTS o CSP")
	}
	if got := e.do("GET", "/api/v1/stats", nil, map[string]string{"Origin": "https://malo.example"}).Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("origen no permitido recibió CORS: %q", got)
	}
	// geobloqueo vía cabecera de CDN solo si el salto es un proxy de confianza
	req := httptest.NewRequest("GET", "/api/v1/stats", nil)
	req.RemoteAddr = "10.1.2.3:5000"
	req.Header.Set("X-Forwarded-For", "198.51.100.9")
	req.Header.Set("CF-IPCountry", "RU")
	r1 := httptest.NewRecorder()
	e.h.ServeHTTP(r1, req)
	if r1.Code != 403 || !strings.Contains(r1.Body.String(), "region_blocked") {
		t.Errorf("RU vía proxy de confianza debe bloquearse: %d", r1.Code)
	}
	if rec := e.do("GET", "/api/v1/stats", nil, map[string]string{"CF-IPCountry": "RU"}); rec.Code != 200 {
		t.Errorf("cabecera falsificada desde un cliente directo no debe bloquear ni confiarse: %d", rec.Code)
	}
}

func TestRateLimiting(t *testing.T) {
	e := setup(t, func(c *config.Config) {
		c.RateAnon = config.Rate{RPS: 1, Burst: 3}
		c.RateKey = config.Rate{RPS: 100, Burst: 100}
	})
	var codes []int
	for i := 0; i < 6; i++ {
		codes = append(codes, e.do("GET", "/api/v1/stats", nil, nil).Code)
	}
	if codes[0] != 200 || codes[5] != 429 {
		t.Fatalf("anónimo: %v", codes)
	}
	rec := e.do("GET", "/api/v1/stats", nil, nil)
	if rec.Header().Get("Retry-After") == "" {
		t.Error("falta Retry-After")
	}
	// con API key válida el cupo es otro; con una inválida se trata como anónimo
	if rec := e.do("GET", "/api/v1/stats", nil, map[string]string{"X-API-Key": "clave-integrador"}); rec.Code != 200 {
		t.Errorf("API key válida: %d", rec.Code)
	}
	if rec := e.do("GET", "/api/v1/stats", nil, map[string]string{"X-API-Key": "inventada"}); rec.Code != 429 {
		t.Errorf("API key inválida no debe saltarse el límite: %d", rec.Code)
	}
}

func TestLimitesDeAdministracion(t *testing.T) {
	e := setup(t, func(c *config.Config) {
		c.RateAdmin = config.Rate{RPS: 0.1, Burst: 2}       // carga: estricta
		c.RateAdminRead = config.Rate{RPS: 100, Burst: 100} // sondeo de estado: holgado
	})
	tok := map[string]string{"Authorization": "Bearer s3cr3to-largo-de-prueba"}
	var codes []int
	for i := 0; i < 4; i++ {
		codes = append(codes, e.do("POST", "/api/v1/admin/schemas?fileName=a.tsp", strings.NewReader("namespace A;"), tok).Code)
	}
	if codes[0] != 202 || codes[3] != 429 {
		t.Errorf("la carga debe limitarse: %v", codes)
	}
	// el sondeo del estado no se ve afectado por el límite de carga
	for i := 0; i < 30; i++ {
		if rec := e.do("GET", "/api/v1/admin/jobs/20250101000000000-abcdef", nil, tok); rec.Code == 429 {
			t.Fatalf("el sondeo de estado no debe limitarse con el cupo de cargas (iteración %d)", i)
		}
	}
}
