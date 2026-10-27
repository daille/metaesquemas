import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { config } from "./config.js";
import { IngestError } from "./validate.js";

const child = fileURLToPath(new URL("./compile-child.js", import.meta.url));

/**
 * Compila `main` dentro de `workspace` en un proceso aparte, con límite de tiempo y de memoria y sin
 * heredar variables de entorno (secretos). Devuelve los diagnósticos o lanza IngestError.
 */
export function compileInChild({ workspace, main, outDir }) {
  return new Promise((resolve, reject) => {
    const p = spawn(process.execPath, [`--max-old-space-size=${config.compileMemoryMb}`, child, JSON.stringify({ workspace, main, outDir })], {
      cwd: workspace,
      env: { PATH: process.env.PATH, HOME: workspace, NODE_ENV: "production" },
      stdio: ["ignore", "pipe", "pipe"],
    });
    let out = "", err = "";
    const timer = setTimeout(() => { p.kill("SIGKILL"); reject(new IngestError(`La compilación superó el límite de ${config.compileTimeoutMs} ms.`)); }, config.compileTimeoutMs);
    p.stdout.on("data", (d) => { out += d; if (out.length > 5e6) p.kill("SIGKILL"); });
    p.stderr.on("data", (d) => { err += d; if (err.length > 1e6) p.kill("SIGKILL"); });
    p.on("error", (e) => { clearTimeout(timer); reject(e); });
    p.on("close", (code, signal) => {
      clearTimeout(timer);
      const m = out.split("@@RESULT@@")[1];
      if (!m) {
        const why = signal === "SIGKILL" ? "proceso terminado (límite de memoria o tiempo)" : err.split("\n").filter(Boolean).slice(0, 3).join(" | ") || `código ${code}`;
        return reject(new IngestError(`Falló la compilación: ${why}`));
      }
      resolve(JSON.parse(m).diagnostics);
    });
  });
}
