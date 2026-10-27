import { mkdir, writeFile, rename } from "node:fs/promises";
import { randomBytes } from "node:crypto";
import { join } from "node:path";
import { config } from "./config.js";

/** Encola un trabajo de ingesta (mismo protocolo que usa la API Go). */
export async function submitJob({ source, fileName, uploader }) {
  const id = new Date().toISOString().replace(/\D/g, "").slice(0, 17) + "-" + randomBytes(3).toString("hex");
  const tmp = join(config.inboxDir, `.tmp-${id}`);
  await mkdir(tmp, { recursive: true });
  await writeFile(join(tmp, "source.tsp"), source);
  await writeFile(join(tmp, "job.json"), JSON.stringify({ id, fileName, uploader: uploader || null, submittedAt: new Date().toISOString() }));
  await mkdir(config.jobsDir, { recursive: true });
  await writeFile(join(config.jobsDir, `${id}.json`), JSON.stringify({ id, status: "queued", fileName, submittedAt: new Date().toISOString() }));
  await rename(tmp, join(config.inboxDir, id)); // visible para el worker solo cuando está completo
  return id;
}
