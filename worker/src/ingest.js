import { mkdir, mkdtemp, readFile, writeFile, symlink, copyFile, readdir } from "node:fs/promises";
import { createHash } from "node:crypto";
import { join, relative, dirname, posix } from "node:path";
import { fileURLToPath } from "node:url";
import { config } from "./config.js";
import { validateSource, resolveImports, IngestError } from "./validate.js";
import { compileInChild } from "./compile.js";
import { bundleJsonSchema } from "./jsonschema.js";
import * as store from "./store.js";

const nodeModules = fileURLToPath(new URL("../node_modules", import.meta.url));
const sha256 = (s) => createHash("sha256").update(s).digest("hex");

async function exists(p) { try { await readFile(p); return true; } catch { return false; } }

/**
 * Compila y publica una versión de un archivo. Todo el trabajo pesado ocurre aquí (al cargar),
 * nunca en la lectura. Devuelve { status: "published" | "unchanged", rel, version, ... }.
 */
async function buildAndPublish({ rel, source, namespace, reason, uploader, force, log }) {
  const hash = sha256(source);
  const current = await store.readCurrent(rel);
  const currentMeta = current ? await store.readMeta(rel, current) : null;
  if (current && currentMeta?.file.sha256 === hash && !force) return { status: "unchanged", rel, version: current };

  await mkdir(config.workDir, { recursive: true });
  const ws = await mkdtemp(join(config.workDir, "ws-"));
  const t0 = Date.now();
  try {
    // Espacio de trabajo: repositorio vigente + el archivo nuevo, con las librerías del worker.
    await store.materializeSources(join(ws, "src"));
    await mkdir(dirname(join(ws, "src", rel)), { recursive: true });
    await writeFile(join(ws, "src", rel), source);
    await symlink(nodeModules, join(ws, "node_modules"));
    await writeFile(join(ws, "package.json"), '{"name":"ws","private":true,"type":"module"}');

    const main = join(ws, "src", rel);
    const outDir = join(ws, "out");
    log?.(`Compilando ${rel}…`);
    const diagnostics = await compileInChild({ workspace: ws, main, outDir });
    const errors = diagnostics.filter((d) => d.severity === "error");
    if (errors.length) throw new IngestError("Errores de compilación TypeSpec.", errors);
    const warnings = diagnostics.filter((d) => d.severity !== "error");

    const meta = JSON.parse(await readFile(join(outDir, "@catalogo/typespec/metadata.json"), "utf8"));
    if (meta.entries.length === 0) throw new IngestError("El archivo no declara ningún esquema (model, scalar, enum, union o interface).");

    // Reglas del catálogo
    const missing = meta.entries.filter((e) => !e.responsable);
    if (missing.length) throw new IngestError("Cada esquema debe tener un responsable: use @responsable(\"Unidad o Persona\") en el tipo o en su namespace.",
      missing.map((e) => ({ message: `Sin responsable: ${e.id}` })));
    const index = await store.readIndex();
    const clash = meta.entries.filter((e) => index.entries.some((x) => x.id === e.id && x.file !== rel));
    if (clash.length) throw new IngestError("Ya existen esquemas con el mismo nombre completo en otro archivo.",
      clash.map((e) => ({ message: `${e.id} ya está definido en ${index.entries.find((x) => x.id === e.id).file}` })));

    // Importaciones transitivas dentro del repositorio
    const root = join(ws, "src");
    const imports = meta.imports.map((p) => relative(root, p)).filter((p) => !p.startsWith("..") && p !== rel).sort();

    // Artefactos
    const n = current + 1;
    const tmpV = join(ws, "version");
    const art = join(tmpV, "artifacts");
    await mkdir(art, { recursive: true });
    await writeFile(join(tmpV, "source.tsp"), source);
    const ts = join(outDir, "@catalogo/typespec");

    const openapis = [];
    try {
      for (const f of await readdir(join(outDir, "@typespec/openapi3"))) {
        const doc = JSON.parse(await readFile(join(outDir, "@typespec/openapi3", f), "utf8"));
        if (doc.paths && Object.keys(doc.paths).length) openapis.push({ file: f, title: doc.info?.title, doc });
      }
    } catch { /* sin servicios */ }

    for (const e of meta.entries) {
      const p = e.projections || {};
      for (const kind of ["xsd", "proto"]) {
        if (p[kind]) { await mkdir(join(art, kind), { recursive: true }); await copyFile(join(ts, p[kind]), join(art, p[kind])); p[kind] = `${kind}/${p[kind].split("/").pop()}`; }
      }
      if (e.kind !== "interface" && !e.plantilla) {
        const bundled = await bundleJsonSchema(join(outDir, "@typespec/json-schema"), e.name, `urn:catalogo:${e.id}`);
        if (bundled) { await mkdir(join(art, "jsonschema"), { recursive: true }); await writeFile(join(art, "jsonschema", `${e.id}.json`), JSON.stringify(bundled, null, 2)); p.jsonschema = `jsonschema/${e.id}.json`; }
      }
      if (e.kind === "interface" && openapis.length) {
        const match = openapis.find((o) => o.title && o.title === e.servicio?.titulo) || (openapis.length === 1 ? openapis[0] : null);
        if (match) {
          await mkdir(join(art, "openapi"), { recursive: true });
          const name = `${match.file.replace(/\.json$/, "")}.json`;
          await writeFile(join(art, "openapi", name), JSON.stringify(match.doc, null, 2));
          p.openapi = `openapi/${name}`;
        }
      }
      e.projections = p;
    }

    const history = [...(currentMeta?.file.history || []), { version: n, createdAt: new Date().toISOString(), motivo: reason }];
    const fileMeta = { path: rel, namespace, version: n, sha256: hash, createdAt: history.at(-1).createdAt, motivo: reason, subidoPor: uploader || null, imports, history };
    const doc = {
      schemaVersion: 1, file: fileMeta,
      build: { compilerVersion: "1.17.0", durationMs: Date.now() - t0, warnings, avisos: meta.avisos || [] },
      entries: meta.entries,
    };
    await writeFile(join(tmpV, "meta.json"), JSON.stringify(doc, null, 2));
    await store.publishVersion(rel, n, tmpV);
    await store.rebuildIndex();
    log?.(`Publicado ${rel} v${n} (${meta.entries.length} esquemas, ${Date.now() - t0} ms)`);
    return { status: "published", rel, version: n, entries: meta.entries.map((e) => e.id), warnings, avisos: meta.avisos || [], imports };
  } finally {
    await store.clean(ws);
  }
}

/**
 * Procesa la carga de un archivo: valida, compila, publica y recompila los archivos que dependen de él.
 */
export async function ingestSource({ source, fileName, uploader, log }) {
  const info = validateSource(source, fileName);
  const rel = posix.join(info.dir, fileName);
  const wanted = resolveImports(rel, info.imports);
  for (const imp of wanted) {
    if (!(await store.readCurrent(imp))) throw new IngestError(`La importación "${imp}" no existe en el repositorio. Suba primero ese archivo. Recuerde: este archivo se guardará en "${posix.dirname(rel)}/" (carpeta derivada de su namespace) y las rutas relativas se resuelven desde allí.`);
  }
  const result = await buildAndPublish({ rel, source, namespace: info.namespace, reason: "carga", uploader, force: false, log });
  result.dependents = result.status === "published" ? await rebuildDependents(rel, log) : [];
  return result;
}

/** Recompila los archivos que importan (directa o transitivamente) a `rel`, para que sus proyecciones no queden obsoletas. */
async function rebuildDependents(rel, log) {
  const index = await store.readIndex();
  const deps = index.files.filter((f) => f.path !== rel && f.imports.includes(rel)).sort((a, b) => a.imports.length - b.imports.length).slice(0, config.maxDependentRebuilds);
  const out = [];
  for (const f of deps) {
    try {
      const n = await store.readCurrent(f.path);
      const source = await store.readSource(f.path, n);
      const r = await buildAndPublish({ rel: f.path, source, namespace: f.namespace, reason: `dependencia actualizada: ${rel}`, uploader: null, force: true, log });
      out.push({ file: f.path, status: r.status, version: r.version });
    } catch (e) {
      // El cambio rompe a un dependiente: se conserva su versión anterior y se informa.
      log?.(`No se pudo recompilar ${f.path}: ${e.message}`);
      out.push({ file: f.path, status: "error", error: e.message, details: e.details || [] });
    }
  }
  return out;
}

export { IngestError };
