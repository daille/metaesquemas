package store

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/org/metaesquemas/internal/domain"
	"github.com/org/metaesquemas/internal/testutil"
)

func newCatalog(t *testing.T) (*Catalog, string) {
	dir := testutil.NewStore(t)
	c, err := New(dir, 40*time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return c, dir
}

func ids(r domain.ListResult) []string {
	var out []string
	for _, e := range r.Items {
		out = append(out, e.ID)
	}
	return out
}

func TestListFiltros(t *testing.T) {
	c, _ := newCatalog(t)
	if got := c.List(domain.ListParams{Limit: 50}); got.Total != 7 {
		t.Fatalf("total=%d", got.Total)
	}
	if got := c.List(domain.ListParams{Limit: 50, Tipos: []string{"basico"}}); got.Total != 2 {
		t.Errorf("tipo basico=%d", got.Total)
	}
	if got := c.List(domain.ListParams{Limit: 50, Tipos: []string{"documental", "servicio"}}); got.Total != 2 {
		t.Errorf("tipos múltiples=%d", got.Total)
	}
	if got := c.List(domain.ListParams{Limit: 50, Responsable: "Registro Civil"}); got.Total != 1 || got.Items[0].ID != "Pub.Doc.Cert" {
		t.Errorf("responsable: %v", ids(got))
	}
	// vigencia calculada contra la fecha de hoy
	for vig, want := range map[string]string{"vencida": "Pub.Doc.Cert", "futura": "Pub.Base.Estado", "sin-vigencia": "Pub.Svc.Api"} {
		if got := c.List(domain.ListParams{Limit: 50, Vigencia: vig}); got.Total != 1 || got.Items[0].ID != want {
			t.Errorf("vigencia %s: %v", vig, ids(got))
		}
	}
	if got := c.List(domain.ListParams{Limit: 50, Vigencia: "vigente"}); got.Total != 4 {
		t.Errorf("vigentes=%d", got.Total)
	}
	// "usada por otros esquemas"
	if got := c.List(domain.ListParams{Limit: 50, Usada: "no"}); got.Total != 2 {
		t.Errorf("no usadas=%v", ids(got)) // Estado y Api
	}
	if got := c.List(domain.ListParams{Limit: 50, Usada: "si"}); got.Total != 5 {
		t.Errorf("usadas=%v", ids(got))
	}
	// búsqueda sin distinguir tildes ni mayúsculas, varias palabras (AND), sobre id, doc y responsable
	if got := c.List(domain.ListParams{Limit: 50, Q: "DIVISION datos"}); got.Total != 3 {
		t.Errorf("q responsable sin tildes: %v", ids(got))
	}
	if got := c.List(domain.ListParams{Limit: 50, Q: "doc de pub.base.rut"}); got.Total != 1 {
		t.Errorf("q doc: %v", ids(got))
	}
	if got := c.List(domain.ListParams{Limit: 50, Namespace: "Pub.Comun"}); got.Total != 3 {
		t.Errorf("namespace: %v", ids(got))
	}
	if got := c.List(domain.ListParams{Limit: 50, Namespace: "Pub"}); got.Total != 7 {
		t.Errorf("namespace raíz: %d", got.Total)
	}
	if got := c.List(domain.ListParams{Limit: 50, Namespace: "Pu"}); got.Total != 0 {
		t.Errorf("el namespace no debe coincidir por prefijo parcial: %d", got.Total)
	}
	// orden por usos y paginación
	s := c.List(domain.ListParams{Limit: 2, Sort: "usos"})
	if s.Items[0].ID != "Pub.Base.Rut" || s.Total != 7 || len(s.Items) != 2 {
		t.Errorf("sort usos: %v", ids(s))
	}
	if p := c.List(domain.ListParams{Limit: 5, Offset: 5}); len(p.Items) != 2 {
		t.Errorf("paginación: %d", len(p.Items))
	}
	if p := c.List(domain.ListParams{Limit: 5, Offset: 99}); len(p.Items) != 0 || p.Items == nil {
		t.Errorf("fuera de rango debe devolver lista vacía no nula")
	}
}

func TestStatsFacetsNamespaces(t *testing.T) {
	c, _ := newCatalog(t)
	st := c.Stats()
	if st.Total != 7 || st.ByTipo["basico"] != 2 || st.ByTipo["servicio"] != 1 || st.Vencidos != 1 || st.Archivos != 4 {
		t.Errorf("stats: %+v", st)
	}
	if len(st.MasUsados) == 0 || st.MasUsados[0].ID != "Pub.Base.Rut" {
		t.Errorf("más usados: %+v", st.MasUsados)
	}
	f := c.Facets()
	if len(f.Tipos) != 4 || len(f.Responsables) != 4 {
		t.Errorf("facets: %+v", f)
	}
	ns := c.Namespaces()
	if len(ns) != 1 || ns[0].Name != "Pub" || ns[0].Count != 7 || len(ns[0].Children) != 4 {
		t.Errorf("namespaces: %+v", ns)
	}
}

func TestArbolDeDependencias(t *testing.T) {
	c, _ := newCatalog(t)
	tr, err := c.Tree("Pub.Svc.Api", "uses", 3)
	if err != nil {
		t.Fatal(err)
	}
	var missing, found bool
	var walk func(n *domain.TreeNode)
	walk = func(n *domain.TreeNode) {
		if n.ID == "Otro.NoExiste" && !n.Exists {
			missing = true
		}
		if n.ID == "Pub.Base.Rut" && n.Relation == "composicion" {
			found = true
		}
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(tr)
	if !missing || !found {
		t.Errorf("árbol incompleto missing=%v found=%v", missing, found)
	}
	// uso inverso
	in, _ := c.Tree("Pub.Base.Rut", "usedBy", 2)
	if len(in.Children) != 3 {
		t.Errorf("Rut debe ser usado por 3 esquemas distintos (no 4 relaciones): %d", len(in.Children))
	}
	for _, ch := range in.Children {
		if ch.ID == "Pub.Comun.Persona" && (ch.Relation != "composicion" || len(ch.Relations) != 2) {
			t.Errorf("las relaciones con un mismo esquema deben agruparse: %+v", ch)
		}
	}
	if d, _ := c.Get("Pub.Base.Rut", 0); d.UsedByCount != 3 || len(d.UsedBy) != 4 {
		t.Errorf("conteo de esquemas distintos: %d (relaciones: %d)", d.UsedByCount, len(d.UsedBy))
	}
	// ciclos: no debe recursar infinitamente y debe marcarlos
	cy, _ := c.Tree("Pub.Comun.Ciclo1", "uses", 6)
	var cycle bool
	var w2 func(n *domain.TreeNode)
	w2 = func(n *domain.TreeNode) {
		cycle = cycle || n.Cycle
		for _, ch := range n.Children {
			w2(ch)
		}
	}
	w2(cy)
	if !cycle {
		t.Error("no se marcó el ciclo")
	}
	if _, err := c.Tree("No.Existe", "uses", 3); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("esperaba ErrNotFound: %v", err)
	}
	// profundidad acotada
	d1, _ := c.Tree("Pub.Svc.Api", "uses", 1)
	for _, ch := range d1.Children {
		if len(ch.Children) != 0 {
			t.Error("depth=1 no debe expandir nietos")
		}
	}
}

func TestGrafo(t *testing.T) {
	c, _ := newCatalog(t)
	g, err := c.Graph("Pub.Comun.Persona", 1)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, n := range g.Nodes {
		have[n.ID] = true
	}
	for _, id := range []string{"Pub.Comun.Persona", "Pub.Base.Rut", "Pub.Doc.Cert", "Pub.Svc.Api"} {
		if !have[id] {
			t.Errorf("falta nodo %s", id)
		}
	}
	for _, e := range g.Edges {
		if !have[e.Source] || !have[e.Target] {
			t.Errorf("arista con extremo inexistente: %+v", e)
		}
	}
	g2, _ := c.Graph("Pub.Comun.Persona", 2)
	if len(g2.Nodes) <= len(g.Nodes) {
		t.Error("depth=2 debe ampliar el vecindario")
	}
}

func TestDetalleYVersiones(t *testing.T) {
	c, _ := newCatalog(t)
	d, err := c.Get("Pub.Comun.Persona", 0)
	if err != nil || !d.IsCurrent || d.Requested != 2 || d.Responsable.Nombre != "División de Datos" {
		t.Fatalf("vigente: %+v %v", d, err)
	}
	if len(d.Versions) != 2 || d.Available["jsonschema"] == "" || d.Available["tsp-file"] == "" || d.Source == nil {
		t.Errorf("detalle incompleto: %+v", d.Available)
	}
	old, err := c.Get("Pub.Comun.Persona", 1)
	if err != nil || old.IsCurrent || old.Responsable.Nombre != "Responsable anterior" {
		t.Fatalf("versión 1: %+v %v", old, err)
	}
	// una versión en la que el esquema no existía
	if _, err := c.Get("Pub.Comun.Ciclo1", 1); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("Ciclo1 no existía en v1: %v", err)
	}
	if _, err := c.Get("Pub.Comun.Persona", 9); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("versión inexistente: %v", err)
	}
	if _, err := c.Get("No.Existe", 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("esquema inexistente: %v", err)
	}
}

func TestArtefactos(t *testing.T) {
	c, _ := newCatalog(t)
	read := func(kind string, ver int) string {
		a, err := c.Artifact("Pub.Comun.Persona", kind, ver)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		defer a.Close()
		b, _ := io.ReadAll(a.Reader)
		return string(b)
	}
	if got := read("jsonschema", 0); got != `{"type":"object"}` {
		t.Errorf("jsonschema: %q", got)
	}
	if got := read("tsp", 0); got != "model Persona {}" {
		t.Errorf("fragmento: %q", got)
	}
	if got := read("tsp-file", 1); got != "// fuente de pub/comun/personas.tsp v1\n" {
		t.Errorf("archivo completo v1: %q", got)
	}
	a, _ := c.Artifact("Pub.Comun.Persona", "jsonschema", 1)
	if !a.Immutable {
		t.Error("una versión explícita debe ser cacheable como inmutable")
	}
	a.Close()
	if _, err := c.Artifact("Pub.Comun.Persona", "xsd", 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("proyección no generada: %v", err)
	}
	if _, err := c.Artifact("Pub.Comun.Persona", "exe", 0); !errors.Is(err, domain.ErrInvalid) {
		t.Errorf("tipo inválido: %v", err)
	}
	// meta.json manipulado con ruta fuera de artifacts/: debe rechazarse
	if _, err := c.Artifact("Pub.Base.Estado", "proto", 2); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("path traversal debe rechazarse: %v", err)
	}
}

func TestRecargaAutomatica(t *testing.T) {
	c, dir := newCatalog(t)
	if c.Stats().Total != 7 {
		t.Fatal("estado inicial")
	}
	b, _ := os.ReadFile(filepath.Join(dir, "index.json"))
	nb := append([]byte(nil), b...)
	// el worker publica un nuevo índice (sin entradas): la API debe recogerlo sola
	os.WriteFile(filepath.Join(dir, "index.json"), []byte(`{"epoch":2,"files":[],"entries":[]}`), 0o644)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && c.Stats().Total != 0 {
		time.Sleep(50 * time.Millisecond)
	}
	if c.Stats().Total != 0 {
		t.Fatal("no recargó el índice")
	}
	os.WriteFile(filepath.Join(dir, "index.json"), nb, 0o644)
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && c.Stats().Total != 7 {
		time.Sleep(50 * time.Millisecond)
	}
	if c.Stats().Total != 7 {
		t.Fatal("no volvió a cargar")
	}
}

func TestColaDeTrabajos(t *testing.T) {
	d := t.TempDir()
	q := NewQueue(filepath.Join(d, "inbox"), filepath.Join(d, "jobs"))
	id, err := q.Submit(context.Background(), "a.tsp", "ana@x", []byte("namespace A;"))
	if err != nil {
		t.Fatal(err)
	}
	if !jobIDRe.MatchString(id) {
		t.Fatalf("id con formato inesperado: %s", id)
	}
	if b, _ := os.ReadFile(filepath.Join(d, "inbox", id, "source.tsp")); string(b) != "namespace A;" {
		t.Error("source.tsp no quedó en la cola")
	}
	if _, err := os.Stat(filepath.Join(d, "inbox", id, "job.json")); err != nil {
		t.Error("falta job.json")
	}
	if entries, _ := os.ReadDir(filepath.Join(d, "inbox")); len(entries) != 1 {
		t.Errorf("no debe quedar el directorio temporal: %v", entries)
	}
	if b, err := q.Job(id); err != nil || len(b) == 0 {
		t.Errorf("estado: %v", err)
	}
	for _, bad := range []string{"../../etc/passwd", "x", "20250101000000000-zzzzzz", ""} {
		if _, err := q.Job(bad); err == nil {
			t.Errorf("id inválido aceptado: %q", bad)
		}
	}
}
