// Package testutil arma un repositorio de esquemas de prueba con el mismo formato que publica el worker.
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func w(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	var b []byte
	switch x := v.(type) {
	case string:
		b = []byte(x)
	default:
		b, _ = json.MarshalIndent(v, "", " ")
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

type M = map[string]any

func resp(n string, inh bool) M { return M{"nombre": n, "heredado": inh} }
func entry(id, ns, kind, tipo, file string, ver int, r M, vig M, proj []string, deps []M, used []M) M {
	return M{"id": id, "name": id[len(ns)+1:], "namespace": ns, "kind": kind, "tipo": tipo, "tipoInferido": true, "doc": "Doc de " + id,
		"responsable": r, "vigencia": vig, "file": file, "version": ver, "projections": proj, "dependencies": deps, "usedBy": used}
}
func dep(id, rel string, ok bool) M { return M{"id": id, "relation": rel, "exists": ok} }
func use(id, rel string) M          { return M{"id": id, "relation": rel} }

// NewStore crea un repositorio con 7 esquemas:
//
//	Pub.Base.Rut (basico, usado por 3) · Pub.Base.Estado (basico, sin uso) · Pub.Comun.Persona (estructurado)
//	Pub.Doc.Cert (documental, vence) · Pub.Svc.Api (servicio, dependencia inexistente) · Pub.Comun.Ciclo1/Ciclo2 (ciclo)
//
// Persona tiene 2 versiones; la 1 tenía otro responsable.
func NewStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	proj := []string{"jsonschema", "xsd", "proto", "tsp"}
	vigOK := M{"desde": "2020-01-01", "hasta": nil, "heredada": false}
	vigOld := M{"desde": "2015-01-01", "hasta": "2016-12-31", "heredada": false}
	vigFut := M{"desde": "2999-01-01", "hasta": nil, "heredada": false}
	entries := []M{
		entry("Pub.Base.Rut", "Pub.Base", "scalar", "basico", "pub/base/tipos.tsp", 1, resp("Unidad Base", false), vigOK, proj, []M{}, []M{use("Pub.Comun.Persona", "composicion"), use("Pub.Comun.Persona", "uso"), use("Pub.Doc.Cert", "composicion"), use("Pub.Svc.Api", "uso")}),
		entry("Pub.Base.Estado", "Pub.Base", "enum", "basico", "pub/base/tipos.tsp", 1, resp("Unidad Base", false), vigFut, proj, []M{}, []M{}),
		entry("Pub.Comun.Persona", "Pub.Comun", "model", "estructurado", "pub/comun/personas.tsp", 2, resp("División de Datos", false), vigOK, proj, []M{dep("Pub.Base.Rut", "composicion", true)}, []M{use("Pub.Doc.Cert", "composicion"), use("Pub.Svc.Api", "retorno")}),
		entry("Pub.Doc.Cert", "Pub.Doc", "model", "documental", "pub/doc/cert.tsp", 1, resp("Registro Civil", false), vigOld, proj, []M{dep("Pub.Comun.Persona", "composicion", true), dep("Pub.Base.Rut", "composicion", true)}, []M{use("Pub.Svc.Api", "retorno")}),
		entry("Pub.Svc.Api", "Pub.Svc", "interface", "servicio", "pub/svc/api.tsp", 1, resp("Interoperabilidad", false), nil, []string{"openapi", "tsp"}, []M{dep("Pub.Doc.Cert", "retorno", true), dep("Pub.Comun.Persona", "retorno", true), dep("Pub.Base.Rut", "uso", true), dep("Otro.NoExiste", "retorno", false)}, []M{}),
		entry("Pub.Comun.Ciclo1", "Pub.Comun", "model", "estructurado", "pub/comun/personas.tsp", 2, resp("División de Datos", false), vigOK, []string{"tsp"}, []M{dep("Pub.Comun.Ciclo2", "composicion", true)}, []M{use("Pub.Comun.Ciclo2", "composicion")}),
		entry("Pub.Comun.Ciclo2", "Pub.Comun", "model", "estructurado", "pub/comun/personas.tsp", 2, resp("División de Datos", false), vigOK, []string{"tsp"}, []M{dep("Pub.Comun.Ciclo1", "composicion", true)}, []M{use("Pub.Comun.Ciclo1", "composicion")}),
	}
	files := []M{
		{"path": "pub/base/tipos.tsp", "namespace": "Pub.Base", "version": 1, "versions": []M{{"version": 1, "createdAt": "2025-01-01T00:00:00Z", "motivo": "carga"}}, "imports": []string{}},
		{"path": "pub/comun/personas.tsp", "namespace": "Pub.Comun", "version": 2, "versions": []M{{"version": 1, "createdAt": "2025-01-01T00:00:00Z", "motivo": "carga"}, {"version": 2, "createdAt": "2025-02-01T00:00:00Z", "motivo": "carga"}}, "imports": []string{"pub/base/tipos.tsp"}},
		{"path": "pub/doc/cert.tsp", "namespace": "Pub.Doc", "version": 1, "versions": []M{{"version": 1, "createdAt": "2025-01-01T00:00:00Z", "motivo": "carga"}}, "imports": []string{}},
		{"path": "pub/svc/api.tsp", "namespace": "Pub.Svc", "version": 1, "versions": []M{{"version": 1, "createdAt": "2025-01-01T00:00:00Z", "motivo": "carga"}}, "imports": []string{}},
	}
	w(t, filepath.Join(dir, "index.json"), M{"schemaVersion": 1, "epoch": 1, "generatedAt": "2025-02-01T00:00:00Z", "files": files, "entries": entries})

	meta := func(file string, ver int, es []M) {
		base := filepath.Join(dir, "files", filepath.FromSlash(file), "v"+itoa(ver))
		w(t, filepath.Join(base, "source.tsp"), "// fuente de "+file+" v"+itoa(ver)+"\n")
		w(t, filepath.Join(base, "meta.json"), M{"file": M{"path": file, "version": ver}, "build": M{"durationMs": 5}, "entries": es})
		w(t, filepath.Join(base, "artifacts", "jsonschema", "x.json"), `{"type":"object"}`)
		w(t, filepath.Join(base, "artifacts", "openapi", "openapi.json"), `{"openapi":"3.0.0"}`)
	}
	me := func(e M, projections M) M {
		out := M{}
		for k, v := range e {
			out[k] = v
		}
		out["projections"] = projections
		out["source"] = M{"text": "model " + e["name"].(string) + " {}", "startLine": 1, "endLine": 1}
		out["members"] = []M{{"name": "campo", "type": "string", "optional": false}}
		return out
	}
	pj := M{"jsonschema": "jsonschema/x.json"}
	meta("pub/base/tipos.tsp", 1, []M{me(entries[0], pj), me(entries[1], pj)})
	old := me(entries[2], pj)
	old["responsable"] = resp("Responsable anterior", false)
	meta("pub/comun/personas.tsp", 1, []M{old})
	meta("pub/comun/personas.tsp", 2, []M{me(entries[2], pj), me(entries[5], M{}), me(entries[6], M{})})
	meta("pub/doc/cert.tsp", 1, []M{me(entries[3], pj)})
	meta("pub/svc/api.tsp", 1, []M{me(entries[4], M{"openapi": "openapi/openapi.json"})})
	// Un meta.json manipulado: intenta salir del directorio de artefactos
	bad := me(entries[1], M{"proto": "../../../../../../../etc/passwd"})
	w(t, filepath.Join(dir, "files", "pub", "base", "tipos.tsp", "v2", "meta.json"), M{"file": M{"version": 2}, "entries": []M{bad}})
	return dir
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}
