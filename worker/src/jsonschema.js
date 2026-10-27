import { readFile } from "node:fs/promises";
import { join } from "node:path";

const REF = /^([A-Za-z0-9_.-]+)\.json$/;

/**
 * El emisor oficial genera un archivo por modelo con $ref a archivos hermanos ("Rut.json").
 * Esto produce un JSON Schema autocontenido: inserta las dependencias en $defs y reescribe los $ref.
 * Devuelve null si el esquema no fue emitido.
 */
export async function bundleJsonSchema(dir, name, id) {
  const load = async (n) => { try { return JSON.parse(await readFile(join(dir, `${n}.json`), "utf8")); } catch { return null; } };
  const root = await load(name);
  if (!root) return null;
  const defs = {};
  const queue = [];
  const rewrite = (node) => {
    if (Array.isArray(node)) return node.map(rewrite);
    if (node && typeof node === "object") {
      const out = {};
      for (const [k, v] of Object.entries(node)) {
        if (k === "$ref" && typeof v === "string" && REF.test(v)) {
          const dep = v.match(REF)[1];
          if (dep !== name && !(dep in defs)) { defs[dep] = null; queue.push(dep); }
          out[k] = dep === name ? "#" : `#/$defs/${dep}`;
        } else if (k === "$id" || k === "$schema") continue;
        else out[k] = rewrite(v);
      }
      return out;
    }
    return node;
  };
  const body = rewrite(root);
  while (queue.length) {
    const n = queue.shift();
    const d = await load(n);
    defs[n] = d ? rewrite(d) : { description: `Esquema ${n} no disponible.` };
  }
  return { $schema: "https://json-schema.org/draft/2020-12/schema", $id: id, ...body, ...(Object.keys(defs).length ? { $defs: defs } : {}) };
}
