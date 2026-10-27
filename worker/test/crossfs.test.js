import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, stat, readdir } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

// Reproduce el despliegue con Docker: el espacio de trabajo (tmpfs) y el repositorio (volumen) están en
// sistemas de archivos distintos, donde rename() falla con EXDEV. Se omite si no hay un segundo dispositivo.
const data = await mkdtemp(join(tmpdir(), "metaesquemas-xfs-"));
let work = null;
try {
  const shm = await mkdtemp("/dev/shm/metaesquemas-work-");
  if ((await stat(shm)).dev !== (await stat(data)).dev) work = shm;
} catch { /* sin /dev/shm */ }

if (work) { process.env.DATA_DIR = data; process.env.WORK_DIR = work; }
const { ingestSource } = work ? await import("../src/ingest.js") : {};
const store = work ? await import("../src/store.js") : null;

test("publica correctamente cuando el espacio de trabajo está en otro sistema de archivos", { skip: !work && "no hay un segundo sistema de archivos" }, async () => {
  const src = await readFile(new URL("../../samples/chile/basicos/tipos.tsp", import.meta.url), "utf8");
  const r = await ingestSource({ source: src, fileName: "tipos.tsp" });
  assert.equal(r.status, "published");
  assert.equal(await store.readCurrent("chile/basicos/tipos.tsp"), 1);
  assert.ok((await store.readIndex()).entries.length > 0);
  // no quedan directorios temporales de preparación dentro del repositorio
  const left = (await readdir(store.fileDir("chile/basicos/tipos.tsp"))).filter((n) => n.startsWith("."));
  assert.deepEqual(left, []);
});
