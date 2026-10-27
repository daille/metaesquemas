#!/usr/bin/env node
// Uso:
//   node src/cli.js ingest archivo.tsp        compila y publica directamente en el repositorio
//   node src/cli.js seed ./samples [--submit] carga un árbol de .tsp respetando el orden de dependencias
//                                             (--submit: los encola para el worker en lugar de compilar aquí)
import { readFile, readdir } from "node:fs/promises";
import { basename, join, posix, relative } from "node:path";
import { mkdir } from "node:fs/promises";
import { config } from "./config.js";
import { ingestSource, IngestError } from "./ingest.js";
import { submitJob } from "./submit.js";
import { validateSource, resolveImports } from "./validate.js";

const [cmd, arg, flag] = process.argv.slice(2);
for (const d of [config.storeDir, config.inboxDir, config.jobsDir, config.workDir]) await mkdir(d, { recursive: true });

async function tspFiles(dir) {
  const out = [];
  for (const e of await readdir(dir, { withFileTypes: true })) {
    if (e.isDirectory()) { if (e.name !== "node_modules" && !e.name.startsWith(".")) out.push(...(await tspFiles(join(dir, e.name)))); }
    else if (e.name.endsWith(".tsp")) out.push(join(dir, e.name));
  }
  return out;
}

async function ingestOne(path, submit) {
  const source = await readFile(path, "utf8");
  if (submit) { const id = await submitJob({ source, fileName: basename(path), uploader: "cli" }); console.log(`encolado ${path} -> ${id}`); return; }
  try {
    const r = await ingestSource({ source, fileName: basename(path), uploader: "cli", log: console.log });
    console.log(`${r.status}: ${r.rel} v${r.version}`);
  } catch (e) {
    if (!(e instanceof IngestError)) throw e;
    console.error(`ERROR en ${path}: ${e.message}`);
    for (const d of e.details || []) console.error(`  - ${d.file ? d.file + ":" + (d.line || "?") + " " : ""}${d.message || JSON.stringify(d)}`);
    process.exitCode = 1;
  }
}

if (cmd === "ingest" && arg) await ingestOne(arg, false);
else if (cmd === "seed" && arg) {
  // Orden topológico por importaciones relativas
  const files = await tspFiles(arg);
  const rels = new Map(files.map((f) => [relative(arg, f).split("\\").join("/"), f]));
  const order = [], seen = new Set();
  const visit = async (rel) => {
    if (seen.has(rel)) return;
    seen.add(rel);
    const info = validateSource(await readFile(rels.get(rel), "utf8"), basename(rel));
    for (const imp of resolveImports(rel, info.imports)) if (rels.has(imp)) await visit(imp);
    order.push(rel);
  };
  for (const rel of [...rels.keys()].sort()) await visit(rel);
  for (const rel of order) { await ingestOne(rels.get(rel), flag === "--submit"); if (flag === "--submit") await new Promise((r) => setTimeout(r, 5)); } // ids de trabajo estrictamente crecientes
} else { console.error("Uso: cli.js ingest <archivo.tsp> | cli.js seed <directorio> [--submit]"); process.exit(2); }
