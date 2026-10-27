import { resolve } from "node:path";

const env = (k, d) => process.env[k] || d;
const root = resolve(env("DATA_DIR", "./data"));

export const config = {
  dataDir: root,
  storeDir: resolve(env("STORE_DIR", `${root}/store`)),   // repositorio de lectura (la API Go lo monta de solo lectura)
  inboxDir: resolve(env("INBOX_DIR", `${root}/inbox`)),   // cola de trabajos (la API escribe, el worker consume)
  jobsDir: resolve(env("JOBS_DIR", `${root}/jobs`)),      // estado de trabajos
  workDir: resolve(env("WORK_DIR", `${root}/work`)),      // espacio temporal de compilación
  pollMs: Number(env("POLL_MS", "1000")),
  maxSourceBytes: Number(env("MAX_SOURCE_BYTES", String(256 * 1024))),
  compileTimeoutMs: Number(env("COMPILE_TIMEOUT_MS", "60000")),
  compileMemoryMb: Number(env("COMPILE_MEMORY_MB", "768")),
  maxDependentRebuilds: Number(env("MAX_DEPENDENT_REBUILDS", "200")),
};

/** Librerías que un archivo subido puede importar. Todo lo demás (incluido JavaScript) se rechaza. */
export const ALLOWED_PACKAGES = new Set([
  "@catalogo/typespec", "@typespec/http", "@typespec/rest", "@typespec/openapi", "@typespec/versioning",
]);
export const FILE_NAME_RE = /^[A-Za-z0-9][A-Za-z0-9._-]{0,80}\.tsp$/;
