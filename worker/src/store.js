import { mkdir, readFile, writeFile, rename, readdir, rm, cp } from "node:fs/promises";
import { join, relative } from "node:path";
import { config } from "./config.js";

/** Escritura atómica: se escribe a un archivo temporal y se renombra. */
export async function writeAtomic(path, content) {
  await mkdir(join(path, ".."), { recursive: true });
  const tmp = `${path}.${process.pid}.${Date.now()}.tmp`;
  await writeFile(tmp, content);
  await rename(tmp, path);
}

const filesRoot = () => join(config.storeDir, "files");
export const fileDir = (rel) => join(filesRoot(), rel);
export const versionDir = (rel, n) => join(fileDir(rel), `v${n}`);

export async function readCurrent(rel) {
  try { return Number((await readFile(join(fileDir(rel), "current"), "utf8")).trim()) || 0; } catch { return 0; }
}
export async function readMeta(rel, n) {
  try { return JSON.parse(await readFile(join(versionDir(rel, n), "meta.json"), "utf8")); } catch { return null; }
}
export async function readSource(rel, n) { return readFile(join(versionDir(rel, n), "source.tsp"), "utf8"); }

/** Lista las rutas relativas (p. ej. "chile/comunes/personas.tsp") de los archivos publicados. */
export async function listFiles() {
  const out = [];
  const walk = async (dir) => {
    let items;
    try { items = await readdir(dir, { withFileTypes: true }); } catch { return; }
    if (items.some((i) => i.isFile() && i.name === "current")) { out.push(relative(filesRoot(), dir)); return; }
    for (const i of items) if (i.isDirectory() && !i.name.startsWith(".")) await walk(join(dir, i.name));
  };
  await walk(filesRoot());
  return out.sort();
}

/** Copia el estado vigente del repositorio (fuentes) a un directorio de trabajo para resolver importaciones. */
export async function materializeSources(destRoot) {
  for (const rel of await listFiles()) {
    const n = await readCurrent(rel);
    if (!n) continue;
    await mkdir(join(destRoot, rel, ".."), { recursive: true });
    await cp(join(versionDir(rel, n), "source.tsp"), join(destRoot, rel));
  }
}

/** Publica una versión ya armada en un directorio temporal: la mueve a su lugar y actualiza `current`. */
export async function publishVersion(rel, n, tmpDir) {
  await mkdir(fileDir(rel), { recursive: true });
  // El espacio de trabajo puede estar en otro sistema de archivos (p. ej. un tmpfs): rename() entre
  // dispositivos falla con EXDEV. Se copia primero DENTRO del repositorio y se renombra allí (atómico).
  const staging = join(fileDir(rel), `.staging-v${n}-${process.pid}-${Date.now()}`);
  try {
    await cp(tmpDir, staging, { recursive: true });
    await rename(staging, versionDir(rel, n));
  } catch (e) {
    await rm(staging, { recursive: true, force: true });
    throw e;
  }
  await writeAtomic(join(fileDir(rel), "current"), String(n));
}

/**
 * Reconstruye store/index.json (catálogo vigente) a partir de la versión actual de cada archivo.
 * Precalcula "usado por" (relación inversa de dependencias) para que la lectura sea inmediata.
 */
export async function rebuildIndex() {
  const files = [];
  const entries = [];
  for (const rel of await listFiles()) {
    const n = await readCurrent(rel);
    const meta = n ? await readMeta(rel, n) : null;
    if (!meta) continue;
    files.push({
      path: rel, namespace: meta.file.namespace, version: n, sha256: meta.file.sha256, updatedAt: meta.file.createdAt,
      imports: meta.file.imports, versions: meta.file.history,
    });
    for (const e of meta.entries) {
      entries.push({
        id: e.id, name: e.name, namespace: e.namespace, kind: e.kind, tipo: e.tipo, tipoInferido: e.tipoInferido,
        doc: e.doc, responsable: e.responsable, vigencia: e.vigencia, deprecado: e.deprecado, servicio: e.servicio, plantilla: e.plantilla,
        file: rel, version: n, projections: Object.keys(e.projections || {}).concat("tsp"),
        dependencies: e.dependencies,
      });
    }
  }
  const byId = new Map(entries.map((e) => [e.id, e]));
  for (const e of entries) {
    e.dependencies = e.dependencies.map((d) => ({ ...d, exists: byId.has(d.id) }));
    e.usedBy = [];
  }
  for (const e of entries) for (const d of e.dependencies) {
    const t = byId.get(d.id);
    if (t && !t.usedBy.some((u) => u.id === e.id && u.relation === d.relation)) t.usedBy.push({ id: e.id, relation: d.relation });
  }
  entries.sort((a, b) => a.id.localeCompare(b.id));
  const prev = await readFile(join(config.storeDir, "index.json"), "utf8").then(JSON.parse, () => ({ epoch: 0 }));
  const index = { schemaVersion: 1, epoch: (prev.epoch || 0) + 1, generatedAt: new Date().toISOString(), files, entries };
  await writeAtomic(join(config.storeDir, "index.json"), JSON.stringify(index));
  return index;
}

export async function readIndex() {
  try { return JSON.parse(await readFile(join(config.storeDir, "index.json"), "utf8")); } catch { return { epoch: 0, files: [], entries: [] }; }
}

export async function clean(dir) { await rm(dir, { recursive: true, force: true }); }
