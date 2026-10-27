import { readdir, readFile, writeFile, rm, mkdir, stat } from "node:fs/promises";
import { join } from "node:path";
import { config } from "./config.js";
import { ingestSource, IngestError } from "./ingest.js";
import { writeAtomic } from "./store.js";

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const jobPath = (id) => join(config.jobsDir, `${id}.json`);

async function updateStatus(id, patch) {
  let cur = {};
  try { cur = JSON.parse(await readFile(jobPath(id), "utf8")); } catch { /* nuevo */ }
  const next = { ...cur, ...patch };
  await writeAtomic(jobPath(id), JSON.stringify(next));
  return next;
}

async function processJob(id) {
  const dir = join(config.inboxDir, id);
  const job = JSON.parse(await readFile(join(dir, "job.json"), "utf8"));
  const source = await readFile(join(dir, "source.tsp"), "utf8");
  const logs = [];
  const log = (m) => { logs.push(`${new Date().toISOString()} ${m}`); console.log(`[${id}] ${m}`); };
  await updateStatus(id, { status: "running", startedAt: new Date().toISOString() });
  try {
    const r = await ingestSource({ source, fileName: job.fileName, uploader: job.uploader, log });
    await updateStatus(id, { status: r.status, finishedAt: new Date().toISOString(), file: r.rel, version: r.version, entries: r.entries,
      warnings: r.warnings, avisos: r.avisos, dependents: r.dependents, log: logs });
  } catch (e) {
    const known = e instanceof IngestError;
    if (!known) console.error(e);
    await updateStatus(id, { status: "failed", finishedAt: new Date().toISOString(), error: known ? e.message : "Error interno del worker (revise los registros del contenedor).",
      // Estado visible solo para administración: incluye el motivo técnico de los fallos inesperados (p. ej. permisos de disco)
      details: known ? e.details : [{ message: `${e.code ? e.code + ": " : ""}${String(e.message).slice(0, 300)}` }], log: logs });
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
}

async function pruneJobs() {
  const files = (await readdir(config.jobsDir).catch(() => [])).sort();
  for (const f of files.slice(0, Math.max(0, files.length - 1000))) await rm(join(config.jobsDir, f), { force: true });
}

async function main() {
  for (const d of [config.storeDir, config.inboxDir, config.jobsDir, config.workDir]) await mkdir(d, { recursive: true });
  console.log(`Worker de ingesta listo. inbox=${config.inboxDir} store=${config.storeDir}`);
  await pruneJobs();
  for (;;) {
    const ids = (await readdir(config.inboxDir)).filter((n) => !n.startsWith(".")).sort();
    if (ids.length) { await processJob(ids[0]); continue; } // un trabajo a la vez: versiones consecutivas y consistentes
    await sleep(config.pollMs);
  }
}
main().catch((e) => { console.error(e); process.exit(1); });
