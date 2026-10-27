# Plataforma de Repositorio y Catálogo de Metaesquemas

Centraliza, versiona, relaciona y publica esquemas de datos (tipos básicos, estructurados, documentos y contratos de
servicios) **definidos una vez en TypeSpec** y **proyectados al cargarse** a JSON Schema, XSD, Protobuf y OpenAPI.
El portal y la API **solo leen** lo ya calculado: nunca se compila al consultar.

## 1. Arquitectura

```
 ESCRITURA (AOT, al cargar)                                      LECTURA (milisegundos)
 ──────────────────────────                                      ──────────────────────
 POST .tsp ─► API Go ─► inbox (cola en disco) ─► WORKER Node ─►  store/ (repositorio)  ◄─ API Go (chi) ◄─ Nginx ◄─ Vue SPA
              (202)      volumen                  · valida                solo lectura       índice en memoria,
                                                  · compila (aislado)                        ETag, artefactos en disco
                                                  · publica atómico
```

| Pieza | Tecnología | Rol |
|---|---|---|
| `worker/` | Node 22 + TypeSpec 1.17 | Valida y compila al cargar; genera y publica las proyecciones. **Sin red.** |
| `worker/packages/catalogo` | TypeSpec lib + emisor | Decoradores `@responsable/@tipo/@vigencia`; extrae metadatos y dependencias; genera **XSD** y **Protobuf** |
| `backend/` | Go + chi (hexagonal) | API REST/JSON de lectura + encolado de cargas. Seguridad perimetral |
| `frontend/` | Vue 3, Vite, Pinia, Router, Cytoscape | Inicio, catálogo con filtros, detalle multitecnología, documentación |
| `store` (volumen) | JSON + archivos | Repositorio precalculado, versiones inmutables |

### Ingesta y pre-compilación
1. **Carga**: `POST /api/v1/admin/schemas` (token) deja el `.tsp` en `inbox/` y responde `202`. La API no compila nada.
2. **Validación estricta** (`validate.js`): sintaxis, namespace obligatorio y solo importaciones permitidas (`@catalogo/typespec`, `@typespec/http|rest|openapi|versioning` y otros `.tsp` del repositorio). **Se rechaza todo JavaScript** y los `extern`: un `.tsp` puede ejecutar JS en el compilador.
3. **Ubicación por namespace**: `Chile.Comunes` → `chile/comunes/archivo.tsp`. Las importaciones relativas se resuelven desde esa carpeta.
4. **Compilación aislada**: proceso hijo con límite de tiempo y memoria y sin variables de entorno, dentro de un contenedor sin red.
5. **Emisores**: oficiales `@typespec/openapi3` y `@typespec/json-schema` (este se reempaqueta en JSON Schema *autocontenido* por esquema); propios del catálogo para metadatos, XSD y Protobuf (no existen emisores oficiales equivalentes).
6. **Reglas del catálogo**: todo esquema debe tener responsable (propio o heredado del namespace); no se permiten nombres completos duplicados entre archivos.
7. **Publicación atómica** de `v<N>/` (inmutable) y reconstrucción de `index.json`, con el uso inverso («usado por») ya calculado.
8. **Versionado**: contenido idéntico → sin versión nueva. Un cambio crea `vN+1`, y los archivos que importan al modificado **se recompilan** (nueva versión con motivo «dependencia actualizada»). Si el cambio rompe a un dependiente, se informa y este conserva su versión anterior.

### Lectura
Índice en memoria con recarga automática cuando el worker publica; búsqueda sin tildes, filtros por tipo, responsable, vigencia (calculada con la fecha de hoy), «usado por» y namespace; árbol de dependencias (con ciclos marcados) y grafo; artefactos servidos desde disco con ETag (versión vigente se revalida; `?version=N` es `immutable`).

```
store/index.json                                   catálogo vigente
store/files/chile/comunes/personas.tsp/current     "2"
store/files/chile/comunes/personas.tsp/v2/{source.tsp, meta.json, artifacts/{jsonschema,xsd,proto,openapi}/…}
```

## 2. Seguridad
Misma cadena que la plataforma semántica: `RealIP → Recoverer → GeoBlock → SecurityHeaders → CORS → Timeout → RateLimit → handler`.

| Capa | Implementación |
|---|---|
| IP real | `X-Forwarded-For` solo desde proxies de confianza (derecha→izquierda); Nginx la sobrescribe |
| Rate limiting | Token bucket por IP (IPv6 /64) o API key; cupos distintos para lectura, carga (estricto) y consulta de estado; Nginx aplica `limit_req` antes |
| Geobloqueo | País (GeoLite2 o cabecera de CDN de confianza) en modo `deny`/`allow`, bloques CIDR y `GEO_FAIL_CLOSED` |
| Cabeceras / CORS | HSTS, CSP (`default-src 'none'` en API; `'self'` en SPA), `X-Frame-Options`, `nosniff`, Referrer/Permissions-Policy; CORS con lista explícita |
| Ingesta | Token en tiempo constante, límite de tamaño, nombre saneado, UTF-8; **Nginx no la expone** (404); límite por IP antes de validar el token |
| Worker | Sin red, sin secretos, sin importar JS, con timeout y memoria acotados, usuario no root |
| Lectura | Identificadores validados y resueltos solo contra el índice; rutas de artefactos verificadas contra salida del directorio |

El geobloqueo por IP es aproximado (VPN lo evaden) y no reemplaza a un CDN/WAF frente a ataques volumétricos.

## 3. Estructura
```
backend/   cmd/api · internal/{domain,application,adapters/store,transport/{httpapi,middleware},config,testutil}
worker/    src/{main,ingest,validate,compile,compile-child,jsonschema,store,submit,cli}.js · packages/catalogo/{lib,src} · test/
frontend/  src/{views,components,stores,api,router,styles,utils} · tests/ · nginx.conf
samples/   chile/{basicos,comunes,documentos,servicios/personas}/*.tsp  (+ package.json para autoría con editor)
docker-compose.yml · docker-compose.hardened.yml (opcional) · .env.example · Makefile
```

## 4. Puesta en marcha
```bash
cp .env.example .env            # defina ADMIN_TOKEN (openssl rand -hex 32) y CORS_ALLOWED_ORIGINS
docker compose up -d --build
docker compose --profile seed run --rm seed      # carga ./samples (en orden de dependencias)
open http://localhost:8080
```
Subir un esquema (desde la red interna, ver `/api` en el portal): `curl -X POST $API/api/v1/admin/schemas -H "Authorization: Bearer $ADMIN_TOKEN" -F file=@personas.tsp`, y consultar `GET /api/v1/admin/jobs/<id>`.
Autoría local con editor: `cd samples && npm i` (instala el compilador y `@catalogo/typespec`), `npm run check`.
Sin Docker: `cd worker && npm i && DATA_DIR=./data node src/cli.js seed ../samples`, luego la API con `STORE_DIR=./data/store go run ./cmd/api`. Si un contenedor falla con `exec …: operation not permitted`, use solo el compose base (el endurecimiento es opcional).

## 5. Esquemas en TypeSpec
```tsp
import "@catalogo/typespec";
using Catalogo;

@responsable("División de Datos", "datos@gobierno.example")   // obligatorio (tipo o namespace)
@vigencia("2024-06-01", "2026-12-31")                         // opcional: AAAA-MM-DD
namespace Chile.Comunes;

@tipo(TipoEsquema.documental)  // opcional; si falta: scalar/enum→basico, model/union→estructurado, interface→servicio
model Certificado { folio: string; titular: Persona; }
```

## 6. Decisiones y límites conocidos
* **Protobuf**: TypeSpec no tiene emisor genérico; se numeran los campos por orden de declaración (se advierte en cada `.proto`). Reordenar campos cambia la numeración: versione el contrato. Los escalares de usuario se aplanan a su tipo base y no generan mensaje.
* **XSD**: cada esquema se genera autocontenido (con sus dependencias) en un único `targetNamespace`; las uniones no literales se aproximan con `xs:choice` o `xs:anyType`.
* **Plantillas genéricas** (`Pagina<T>`) se catalogan pero no se proyectan hasta instanciarse.
* Un solo worker procesa la cola en serie (versiones consecutivas y consistentes); para más volumen habría que particionar por archivo.
* La lectura es pública; la ingesta exige token y no se expone por Nginx.

## 7. Verificación realizada
* **Worker**: 6 pruebas de integración con el compilador TypeSpec real (validación, metadatos, proyecciones, versionado, recompilación de dependientes, rechazos).
* **Backend**: compila, `go vet` limpio y pruebas del almacén (filtros, árbol con ciclos, versiones, path traversal, recarga) y de la API HTTP (lectura, carga, seguridad, rate limit).
* **Frontend**: build y 10 pruebas de vistas con respuestas reales de la API como fixtures.
* **Extremo a extremo**: carga por la API → worker → lectura por la API; y Nginx real sirviendo la SPA y haciendo proxy a la API.
* **No verificado**: ejecución con Docker/Compose y aspecto visual en navegador (no disponibles en el entorno de desarrollo); `go.sum` no se incluye (`go mod tidy`).
