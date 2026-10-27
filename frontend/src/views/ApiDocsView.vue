<script setup>
import { ref, computed } from 'vue'
import CodeBlock from '../components/CodeBlock.vue'

const origin = location.origin
const api = computed(() => `${origin}/api/v1`)
const lang = ref('curl')
const langs = [{ id: 'curl', l: 'cURL' }, { id: 'go', l: 'Go' }, { id: 'python', l: 'Python' }, { id: 'js', l: 'JavaScript' }]

const endpoints = [
  ['GET', '/stats', 'Totales por tipo de esquema, vigentes y más reutilizados.', '—'],
  ['GET', '/facets', 'Conteos para los filtros: tipos, responsables, vigencia y uso.', '—'],
  ['GET', '/namespaces', 'Árbol de namespaces (carpetas del repositorio) con cantidad de esquemas.', '—'],
  ['GET', '/schemas', 'Listado paginado con filtros y búsqueda de texto.', 'q, tipo, responsable, vigencia, usada, namespace, sort, limit, offset'],
  ['GET', '/schemas/{id}', 'Detalle: metadatos, responsable, estructura, versiones y URLs de cada proyección.', 'version'],
  ['GET', '/schemas/{id}/artifacts/{kind}', 'Contenido de una proyección ya generada. kind: tsp, tsp-file, jsonschema, xsd, proto, openapi.', 'version, download=1'],
  ['GET', '/schemas/{id}/dependencies', 'Árbol de dependencias (uses) o de uso inverso (usedBy).', 'direction, depth (1-6)'],
  ['GET', '/schemas/{id}/graph', 'Nodos y aristas del vecindario, para visualizar la composición.', 'depth (1-3)'],
  ['POST', '/admin/schemas', 'Carga un archivo .tsp: se encola para validarlo, compilarlo y publicarlo. Requiere token.', 'multipart «file»'],
  ['GET', '/admin/jobs/{id}', 'Estado de una carga (queued, running, published, unchanged, failed). Requiere token.', '—'],
]
const examples = computed(() => ({
  curl: `# Buscar esquemas documentales vigentes
curl -s "${api.value}/schemas?tipo=documental&vigencia=vigente&limit=10"

# Detalle de un esquema (incluye URLs de cada proyección)
curl -s "${api.value}/schemas/Chile.Comunes.Persona"

# Descargar sus proyecciones, ya generadas
curl -s "${api.value}/schemas/Chile.Comunes.Persona/artifacts/jsonschema" -o persona.schema.json
curl -s "${api.value}/schemas/Chile.Comunes.Persona/artifacts/xsd"        -o persona.xsd
curl -s "${api.value}/schemas/Chile.Comunes.Persona/artifacts/proto"      -o persona.proto

# OpenAPI de un servicio, y una versión anterior de un esquema
curl -s "${api.value}/schemas/Chile.Servicios.Personas.Personas/artifacts/openapi"
curl -s "${api.value}/schemas/Chile.Comunes.Persona?version=1"

# ¿Quién usa a Rut?
curl -s "${api.value}/schemas/Chile.Basicos.Rut/dependencies?direction=usedBy&depth=2"

# Con API key (cupo ampliado)
curl -s -H "X-API-Key: $CATALOGO_API_KEY" "${api.value}/stats"`,
  go: `package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const api = "${api.value}"

type Page struct {
	Total int \`json:"total"\`
	Items []struct {
		ID          string \`json:"id"\`
		Tipo        string \`json:"tipo"\`
		UsedByCount int    \`json:"usedByCount"\`
		Responsable struct{ Nombre string } \`json:"responsable"\`
	} \`json:"items"\`
}

func main() {
	c := &http.Client{Timeout: 10 * time.Second}
	q := url.Values{"tipo": {"estructurado"}, "usada": {"si"}, "sort": {"usos"}, "limit": {"5"}}
	resp, err := c.Get(api + "/schemas?" + q.Encode())
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		fmt.Println("límite alcanzado; reintente en", resp.Header.Get("Retry-After"), "s")
		return
	}
	var p Page
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		panic(err)
	}
	for _, e := range p.Items {
		fmt.Printf("%-45s usado por %d · %s\\n", e.ID, e.UsedByCount, e.Responsable.Nombre)
	}
}`,
  python: `import os
import time
import requests

API = "${api.value}"
s = requests.Session()
if key := os.getenv("CATALOGO_API_KEY"):
    s.headers["X-API-Key"] = key


def get(path, **params):
    """GET con reintento respetando Retry-After (429/503)."""
    for _ in range(3):
        r = s.get(f"{API}{path}", params=params, timeout=10)
        if r.status_code in (429, 503):
            time.sleep(int(r.headers.get("Retry-After", "2")))
            continue
        r.raise_for_status()
        return r
    raise RuntimeError("servicio no disponible")


# Esquemas de un responsable
for e in get("/schemas", responsable="Registro Civil e Identificación").json()["items"]:
    print(e["id"], e["vigenciaEstado"])

# Descarga el JSON Schema autocontenido de un esquema
detalle = get("/schemas/Chile.Comunes.Persona").json()
schema = get(detalle["available"]["jsonschema"].replace("/api/v1", "")).json()
print(list(schema["properties"]))`,
  js: `const API = "${api.value}";

async function get(path, params = {}) {
  const url = new URL(API + path, location.origin);
  Object.entries(params).forEach(([k, v]) => url.searchParams.set(k, v));
  const res = await fetch(url, { headers: { Accept: "application/json" } });
  if (res.status === 429) throw new Error("Límite alcanzado; reintente en " + res.headers.get("Retry-After") + " s");
  if (!res.ok) throw new Error((await res.json()).error.message);
  return res;
}

// Servicios vigentes y su contrato OpenAPI
const { items } = await (await get("/schemas", { tipo: "servicio", vigencia: "vigente" })).json();
for (const s of items) {
  const openapi = await (await get(\`/schemas/\${s.id}/artifacts/openapi\`)).json();
  console.log(s.id, Object.keys(openapi.paths));
}`,
}))
const upload = computed(() => `# Ingesta (solo red interna / bastión; requiere el token de administración)
curl -X POST "$API_INTERNA/api/v1/admin/schemas" \\
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "X-Uploader: nombre@institucion.cl" \\
  -F "file=@personas.tsp"
# → 202 {"jobId":"20261025103015123-a1b2c3","status":"queued", ...}

curl -s -H "Authorization: Bearer $ADMIN_TOKEN" "$API_INTERNA/api/v1/admin/jobs/20261025103015123-a1b2c3"
# → {"status":"published","file":"chile/comunes/personas.tsp","version":2,"dependents":[...]}`)
const codes = [
  ['400', 'invalid_request', 'Parámetro inválido (tipo, vigencia, versión, nombre de archivo, tamaño…).'],
  ['401', 'unauthorized', 'Falta el token de administración o es incorrecto.'],
  ['403', 'region_blocked', 'El origen de la solicitud está bloqueado por política de geolocalización.'],
  ['404', 'not_found', 'El esquema, la versión o la proyección no existe.'],
  ['429', 'rate_limited', 'Superó el límite de solicitudes. Espere el tiempo de Retry-After.'],
]
</script>

<template>
  <div class="container page doc">
    <h1 class="page-title">Ayuda y documentación de la API</h1>
    <p class="page-subtitle">Lea los esquemas y sus proyecciones desde sus sistemas. La API es de solo lectura sobre un repositorio precalculado: nada se compila al consultar.</p>

    <nav class="toc card" aria-label="Contenido de esta página">
      <a href="#inicio">Primeros pasos</a><a href="#endpoints">Endpoints</a><a href="#ejemplos">Ejemplos</a><a href="#ingesta">Ingesta de esquemas</a><a href="#limites">Límites y errores</a>
    </nav>

    <section id="inicio" class="sec">
      <h2>Primeros pasos</h2>
      <p>La URL base es <code class="mono">{{ api }}</code>. Las respuestas son JSON (UTF-8); las proyecciones se entregan en su formato nativo (<code class="mono">application/json</code>, <code class="mono">application/xml</code> o <code class="mono">text/plain</code>).</p>
      <p>El identificador <code class="mono">{id}</code> de un esquema es su nombre completo: <code class="mono">Namespace.Nombre</code>, por ejemplo <code class="mono">Chile.Comunes.Persona</code>. Las versiones de un archivo son inmutables, por lo que <code class="mono">?version=N</code> es cacheable indefinidamente; la versión vigente se revalida con ETag.</p>
    </section>

    <section id="endpoints" class="sec">
      <h2>Endpoints</h2>
      <div class="table-wrap">
        <table>
          <caption class="sr-only">Endpoints disponibles</caption>
          <thead><tr><th>Método</th><th>Ruta</th><th>Descripción</th><th>Parámetros</th></tr></thead>
          <tbody><tr v-for="e in endpoints" :key="e[1]"><td><span class="badge" :class="e[0] === 'GET' ? 'badge-accent' : 'badge-amber'">{{ e[0] }}</span></td><td class="mono text-sm brk">{{ e[1] }}</td><td>{{ e[2] }}</td><td class="mono text-xs">{{ e[3] }}</td></tr></tbody>
        </table>
      </div>
    </section>

    <section id="ejemplos" class="sec">
      <h2>Ejemplos de uso</h2>
      <div class="tabs" role="tablist" aria-label="Lenguaje del ejemplo"><button v-for="l in langs" :key="l.id" class="tab" role="tab" :aria-selected="lang === l.id" @click="lang = l.id">{{ l.l }}</button></div>
      <CodeBlock :code="examples[lang]" :label="langs.find((l) => l.id === lang).l" />
    </section>

    <section id="ingesta" class="sec">
      <h2>Ingesta de esquemas</h2>
      <p>Los esquemas se escriben en TypeSpec (<code class="mono">.tsp</code>) y se cargan por la API de administración. La carga solo encola el archivo: un <strong>worker</strong> lo valida, lo compila y publica sus proyecciones antes de que aparezca en el catálogo.</p>
      <ul class="bul">
        <li>El archivo debe declarar un <code class="mono">namespace</code>. Se guarda en la carpeta que corresponde a él (<code class="mono">Chile.Comunes</code> → <code class="mono">chile/comunes/</code>); las importaciones relativas se resuelven desde esa carpeta.</li>
        <li>Cada esquema debe declarar su responsable con <code class="mono">@responsable("Unidad o Persona")</code>, en el tipo o en su namespace. También admite <code class="mono">@tipo(TipoEsquema.documental)</code> y <code class="mono">@vigencia("2025-01-01", "2026-12-31")</code>.</li>
        <li>Solo se permiten importaciones de <code class="mono">@catalogo/typespec</code>, <code class="mono">@typespec/http</code>, <code class="mono">/rest</code>, <code class="mono">/openapi</code>, <code class="mono">/versioning</code> y de otros archivos <code class="mono">.tsp</code> del repositorio. Se rechaza todo JavaScript.</li>
        <li>Si el contenido no cambió, no se crea una versión. Si cambia, los archivos que lo importan se recompilan para que sus proyecciones no queden obsoletas.</li>
      </ul>
      <CodeBlock :code="upload" label="cURL" />
    </section>

    <section id="limites" class="sec">
      <h2>Límites, errores y buenas prácticas</h2>
      <div class="table-wrap">
        <table>
          <caption class="sr-only">Códigos de respuesta</caption>
          <thead><tr><th>HTTP</th><th>Código</th><th>Significado</th></tr></thead>
          <tbody><tr v-for="c in codes" :key="c[0]"><td><span class="badge badge-code">{{ c[0] }}</span></td><td class="mono text-sm">{{ c[1] }}</td><td>{{ c[2] }}</td></tr></tbody>
        </table>
      </div>
      <ul class="bul" style="margin-top: 16px">
        <li>Los límites se aplican por dirección IP o por API key (cabecera <code class="mono">X-API-Key</code>). Las cabeceras <code class="mono">X-RateLimit-*</code> y <code class="mono">Retry-After</code> indican su cupo.</li>
        <li>Respete los ETag y <code class="mono">Cache-Control</code>: las proyecciones cambian solo cuando se carga una nueva versión del archivo fuente.</li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.doc { max-width: 960px; }
.toc { display: flex; flex-wrap: wrap; gap: 8px 22px; margin: 24px 0 8px; font-size: 15px; padding: 14px 20px; }
.sec { margin-top: 44px; display: flex; flex-direction: column; gap: 12px; }
.sec h2 { font-size: 22px; letter-spacing: -.02em; }
.sec p { max-width: 74ch; }
code { background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 4px; padding: 0 5px; font-size: 13px; }
.brk { word-break: break-all; }
.bul { padding-left: 20px; display: flex; flex-direction: column; gap: 8px; max-width: 74ch; }
</style>
