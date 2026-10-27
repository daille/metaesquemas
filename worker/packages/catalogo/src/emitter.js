import { emitFile, resolvePath } from "@typespec/compiler";
import { collectEntries } from "./entries.js";
import { generateXsd } from "./xsd.js";
import { generateProto } from "./proto.js";

/** Emisor del catálogo: metadatos + XSD + Protobuf. Se ejecuta junto con openapi3 y json-schema. */
export async function $onEmit(context) {
  const { program, options } = context;
  const mainFile = options["main-file"];
  if (!mainFile) throw new Error("Falta la opción main-file del emisor @catalogo/typespec");
  const main = resolvePath(mainFile);

  const entries = collectEntries(program, main);
  const out = (p) => resolvePath(context.emitterOutputDir, p);
  const projections = {};
  const avisos = [];   // fallos de una proyección no deben impedir publicar el resto
  for (const e of entries) {
    const p = {};
    if (e.plantilla) { projections[e.id] = p; continue; }   // las plantillas genéricas no se proyectan
    let xsd = null, proto = null;
    try { xsd = generateXsd(program, e.type); } catch (err) { avisos.push({ id: e.id, proyeccion: "xsd", mensaje: String(err?.stack || err) }); }
    if (xsd) { await emitFile(program, { path: out(`xsd/${e.id}.xsd`), content: xsd }); p.xsd = `xsd/${e.id}.xsd`; }
    try { proto = generateProto(program, e.type); } catch (err) { avisos.push({ id: e.id, proyeccion: "proto", mensaje: String(err?.stack || err) }); }
    if (proto) { await emitFile(program, { path: out(`proto/${e.id}.proto`), content: proto }); p.proto = `proto/${e.id}.proto`; }
    projections[e.id] = p;
  }

  // Archivos del repositorio importados (transitivamente): sirven para recompilar dependientes.
  const imports = [...program.sourceFiles.keys()]
    .filter((p) => p.endsWith(".tsp") && !p.includes("/node_modules/") && p !== main)
    .sort();

  const meta = {
    schemaVersion: 1,
    mainFile: main,
    imports,
    avisos,
    entries: entries.map(({ type, ...rest }) => ({ ...rest, projections: projections[rest.id] })),
  };
  await emitFile(program, { path: out("metadata.json"), content: JSON.stringify(meta, null, 2) });
}
