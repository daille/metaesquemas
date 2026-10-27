// Package store lee el repositorio de esquemas pre-calculados que publica el worker.
//
// Layout (ver worker/src/store.js):
//
//	store/index.json                                catálogo vigente (resúmenes + "usado por")
//	store/files/<namespace>/<archivo>.tsp/current   número de versión vigente
//	store/files/.../v<N>/{source.tsp,meta.json,artifacts/...}   versión inmutable
package store

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/org/metaesquemas/internal/domain"
)

type snapshot struct {
	idx   domain.Index
	byID  map[string]*domain.Entry
	hay   map[string]string // texto de búsqueda (sin tildes, minúsculas)
	files map[string]*domain.FileInfo
	mtime time.Time
	size  int64
}

type Catalog struct {
	dir   string
	log   *slog.Logger
	snap  atomic.Pointer[snapshot]
	metas sync.Map // clave "archivo@v" -> *meta (las versiones son inmutables)
}

var folder = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ñ", "n")

func fold(s string) string { return folder.Replace(strings.ToLower(s)) }

// New carga el índice y lanza un observador que lo recarga cuando el worker publica cambios.
func New(dir string, every time.Duration, log *slog.Logger) (*Catalog, error) {
	c := &Catalog{dir: dir, log: log}
	if err := c.reload(); err != nil {
		log.Warn("índice aún no disponible; se esperará al worker", "err", err)
		c.snap.Store(build(domain.Index{}, time.Time{}, 0))
	}
	go func() {
		for range time.Tick(every) {
			fi, err := os.Stat(filepath.Join(c.dir, "index.json"))
			s := c.snap.Load()
			if err != nil || (fi.ModTime().Equal(s.mtime) && fi.Size() == s.size) {
				continue
			}
			if err := c.reload(); err != nil {
				log.Error("no se pudo recargar el índice", "err", err)
			}
		}
	}()
	return c, nil
}

func (c *Catalog) reload() error {
	path := filepath.Join(c.dir, "index.json")
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var idx domain.Index
	if err := json.Unmarshal(b, &idx); err != nil {
		return fmt.Errorf("index.json inválido: %w", err)
	}
	c.snap.Store(build(idx, fi.ModTime(), fi.Size()))
	c.log.Info("índice cargado", "epoch", idx.Epoch, "esquemas", len(idx.Entries), "archivos", len(idx.Files))
	return nil
}

func build(idx domain.Index, mt time.Time, size int64) *snapshot {
	s := &snapshot{idx: idx, byID: map[string]*domain.Entry{}, hay: map[string]string{}, files: map[string]*domain.FileInfo{}, mtime: mt, size: size}
	for i := range s.idx.Entries {
		e := &s.idx.Entries[i]
		if e.Dependencies == nil {
			e.Dependencies = []domain.Dep{}
		}
		if e.UsedBy == nil {
			e.UsedBy = []domain.Use{}
		}
		e.UsedByCount, e.DependencyCount = distinctUses(e.UsedBy), distinctDeps(e.Dependencies)
		s.byID[e.ID] = e
		resp := ""
		if e.Responsable != nil {
			resp = e.Responsable.Nombre
		}
		s.hay[e.ID] = fold(strings.Join([]string{e.ID, e.Doc, resp, e.Tipo, e.Kind, e.File}, " "))
	}
	for i := range s.idx.Files {
		s.files[s.idx.Files[i].Path] = &s.idx.Files[i]
	}
	return s
}

// estadoVigencia: vigente | vencida | futura | sin-vigencia, según la fecha de hoy (UTC).
func estadoVigencia(v *domain.Vigencia, today string) string {
	switch {
	case v == nil:
		return "sin-vigencia"
	case today < v.Desde:
		return "futura"
	case v.Hasta != nil && today > *v.Hasta:
		return "vencida"
	}
	return "vigente"
}

func today() string { return time.Now().UTC().Format("2006-01-02") }

func (c *Catalog) withState(e domain.Entry) domain.Entry {
	e.VigenciaEstado = estadoVigencia(e.Vigencia, today())
	return e
}

func sortedKeys(m map[string]int) []domain.Facet {
	out := make([]domain.Facet, 0, len(m))
	for k, v := range m {
		out = append(out, domain.Facet{Value: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// Un esquema puede relacionarse con otro de varias formas (p. ej. composición y retorno): se cuentan esquemas distintos.
func distinctUses(u []domain.Use) int {
	seen := map[string]bool{}
	for _, x := range u {
		seen[x.ID] = true
	}
	return len(seen)
}

func distinctDeps(d []domain.Dep) int {
	seen := map[string]bool{}
	for _, x := range d {
		seen[x.ID] = true
	}
	return len(seen)
}
