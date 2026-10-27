// Proceso hijo: compila un archivo TypeSpec con los emisores. Se ejecuta aislado (tiempo y memoria acotados).
import { compile, NodeHost, getSourceLocation } from "@typespec/compiler";
import { relative } from "node:path";

const { workspace, main, outDir } = JSON.parse(process.argv[2]);

const program = await compile(NodeHost, main, {
  outputDir: outDir,
  emit: ["@typespec/openapi3", "@typespec/json-schema", "@catalogo/typespec"],
  options: {
    "@typespec/json-schema": { "file-type": "json", emitAllModels: true, emitAllRefs: true },
    "@typespec/openapi3": { "file-type": "json" },
    "@catalogo/typespec": { "main-file": main },
  },
});

const diagnostics = program.diagnostics.map((d) => {
  let file, line;
  try {
    const loc = getSourceLocation(d.target, { locateId: true });
    if (loc?.file) {
      file = relative(workspace, loc.file.path);
      line = loc.file.getLineAndCharacterOfPosition(loc.pos).line + 1;
    }
  } catch { /* sin ubicación */ }
  return { severity: d.severity, code: d.code, message: d.message, file, line };
});
process.stdout.write("\n@@RESULT@@" + JSON.stringify({ diagnostics }) + "\n");
