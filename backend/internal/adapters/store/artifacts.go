package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/org/metaesquemas/internal/domain"
)

type metaEntry struct {
	ID          string            `json:"id"`
	Extends     string            `json:"extends"`
	BaseType    string            `json:"baseType"`
	Builtin     string            `json:"builtin"`
	Constraints json.RawMessage   `json:"constraints"`
	Members     json.RawMessage   `json:"members"`
	Values      json.RawMessage   `json:"values"`
	Variants    json.RawMessage   `json:"variants"`
	Operations  json.RawMessage   `json:"operations"`
	Source      *domain.Snippet   `json:"source"`
	Projections map[string]string `json:"projections"`
	// Campos de la entrada, por si se pide una versión que ya no es la vigente
	Entry domain.Entry `json:"-"`
}

type meta struct {
	File    domain.FileInfo `json:"file"`
	Build   json.RawMessage `json:"build"`
	Entries []json.RawMessage
	byID    map[string]*metaEntry
}

// readMeta lee (y cachea: las versiones son inmutables) el meta.json de una versión de un archivo.
func (c *Catalog) readMeta(file string, version int) (*meta, error) {
	key := fmt.Sprintf("%s@%d", file, version)
	if v, ok := c.metas.Load(key); ok {
		return v.(*meta), nil
	}
	if strings.Contains(file, "..") || version < 1 {
		return nil, domain.ErrNotFound
	}
	b, err := os.ReadFile(filepath.Join(c.dir, "files", filepath.FromSlash(file), fmt.Sprintf("v%d", version), "meta.json"))
	if err != nil {
		return nil, domain.ErrNotFound
	}
	var raw struct {
		File    domain.FileInfo   `json:"file"`
		Build   json.RawMessage   `json:"build"`
		Entries []json.RawMessage `json:"entries"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("meta.json inválido en %s: %w", key, err)
	}
	m := &meta{File: raw.File, Build: raw.Build, byID: map[string]*metaEntry{}}
	for _, r := range raw.Entries {
		var me metaEntry
		if err := json.Unmarshal(r, &me); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(r, &me.Entry)
		m.byID[me.ID] = &me
	}
	c.metas.Store(key, m)
	return m, nil
}

func (c *Catalog) Get(id string, version int) (*domain.Detail, error) {
	s := c.snap.Load()
	cur := s.byID[id]
	if cur == nil {
		return nil, domain.ErrNotFound
	}
	fi := s.files[cur.File]
	if version == 0 {
		version = cur.Version
	}
	m, err := c.readMeta(cur.File, version)
	if err != nil {
		return nil, err
	}
	me := m.byID[id]
	if me == nil {
		return nil, domain.ErrNotFound // el esquema no existía en esa versión del archivo
	}
	base := *cur
	isCurrent := version == cur.Version
	if !isCurrent { // la versión pedida puede diferir en responsable, vigencia, etc.
		base = me.Entry
		base.File, base.Version = cur.File, version
		base.UsedBy, base.Dependencies = []domain.Use{}, withExists(s, me.Entry.Dependencies)
		base.UsedByCount, base.DependencyCount = 0, len(base.Dependencies)
		base.Projections = nil
	}
	base.VigenciaEstado = estadoVigencia(base.Vigencia, today())

	available := map[string]string{"tsp": art(id, "tsp", version), "tsp-file": art(id, "tsp-file", version)}
	for kind, p := range me.Projections {
		if p != "" {
			available[kind] = art(id, kind, version)
		}
	}
	d := &domain.Detail{
		Entry: base, Extends: me.Extends, BaseType: me.BaseType, Builtin: me.Builtin, Constraints: me.Constraints, Members: me.Members,
		Values: me.Values, Variants: me.Variants, Operations: me.Operations, Source: me.Source, Available: available,
		Requested: version, IsCurrent: isCurrent, Build: m.Build, FileInfo: fi,
	}
	if fi != nil {
		d.Versions = fi.Versions
	} else {
		d.Versions = []domain.VersionRef{}
	}
	d.Projections = keys(available)
	return d, nil
}

func withExists(s *snapshot, deps []domain.Dep) []domain.Dep {
	out := make([]domain.Dep, len(deps))
	for i, d := range deps {
		d.Exists = s.byID[d.ID] != nil
		out[i] = d
	}
	return out
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func art(id, kind string, version int) string {
	return fmt.Sprintf("/api/v1/schemas/%s/artifacts/%s?version=%d", id, kind, version)
}

var contentTypes = map[string]string{
	"tsp": "text/plain; charset=utf-8", "tsp-file": "text/plain; charset=utf-8", "proto": "text/plain; charset=utf-8",
	"jsonschema": "application/json; charset=utf-8", "openapi": "application/json; charset=utf-8", "xsd": "application/xml; charset=utf-8",
}
var extensions = map[string]string{"tsp": "tsp", "tsp-file": "tsp", "proto": "proto", "jsonschema": "schema.json", "openapi": "openapi.json", "xsd": "xsd"}

type stringReader struct{ *strings.Reader }

func (c *Catalog) Artifact(id, kind string, version int) (domain.Artifact, error) {
	var zero domain.Artifact
	ct, ok := contentTypes[kind]
	if !ok {
		return zero, domain.ErrInvalid
	}
	s := c.snap.Load()
	cur := s.byID[id]
	if cur == nil {
		return zero, domain.ErrNotFound
	}
	explicit := version != 0
	if version == 0 {
		version = cur.Version
	}
	m, err := c.readMeta(cur.File, version)
	if err != nil {
		return zero, err
	}
	me := m.byID[id]
	if me == nil {
		return zero, domain.ErrNotFound
	}
	vdir := filepath.Join(c.dir, "files", filepath.FromSlash(cur.File), fmt.Sprintf("v%d", version))
	name := id + "." + extensions[kind]
	switch kind {
	case "tsp": // fragmento del esquema, ya extraído al cargar
		if me.Source == nil {
			return zero, domain.ErrNotFound
		}
		r := strings.NewReader(me.Source.Text)
		return domain.Artifact{Reader: r, Close: func() error { return nil }, ContentType: ct, FileName: name, ETag: fmt.Sprintf(`W/"%s-%d-tsp"`, id, version), Immutable: explicit}, nil
	case "tsp-file":
		return openFile(filepath.Join(vdir, "source.tsp"), ct, filepath.Base(cur.File), explicit)
	default:
		p := me.Projections[kind]
		if p == "" {
			return zero, domain.ErrNotFound
		}
		full := filepath.Join(vdir, "artifacts", filepath.FromSlash(p))
		if !strings.HasPrefix(filepath.Clean(full), filepath.Join(vdir, "artifacts")+string(filepath.Separator)) {
			return zero, domain.ErrNotFound // defensa contra rutas manipuladas en meta.json
		}
		return openFile(full, ct, name, explicit)
	}
}

func openFile(path, ct, name string, explicit bool) (domain.Artifact, error) {
	f, err := os.Open(path)
	if err != nil {
		return domain.Artifact{}, domain.ErrNotFound
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return domain.Artifact{}, err
	}
	return domain.Artifact{Reader: f, Close: f.Close, ContentType: ct, FileName: name, ModTime: fi.ModTime().Truncate(time.Second),
		ETag: fmt.Sprintf(`W/"%x-%x"`, fi.Size(), fi.ModTime().UnixNano()), Immutable: explicit}, nil
}
