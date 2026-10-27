import { posix } from "node:path";
import { parse, SyntaxKind } from "@typespec/compiler/ast";
import { ALLOWED_PACKAGES, FILE_NAME_RE, config } from "./config.js";

export class IngestError extends Error {
  constructor(message, details = []) { super(message); this.details = details; }
}

/** Convierte "Chile.Comunes" en la ruta de carpeta "chile/comunes". */
export const namespaceToDir = (ns) => ns.split(".").map((s) => s.toLowerCase()).join("/");

/** `namespace A.B.C` se representa como namespaces anidados (A > B > C): se reconstruye el nombre completo. */
function namespaceName(stmt) {
  const parts = [];
  for (let cur = stmt; cur && cur.kind === SyntaxKind.NamespaceStatement; cur = Array.isArray(cur.statements) ? null : cur.statements) parts.push(cur.id.sv);
  return parts.join(".");
}

/**
 * Valida un archivo .tsp ANTES de compilarlo. Un .tsp puede importar JavaScript y ejecutarlo en el
 * compilador, por lo que solo se admiten importaciones de librerías permitidas y de otros .tsp
 * del repositorio (rutas relativas que no salgan de él).
 * Devuelve { namespace, dir, imports: [ruta relativa al archivo] }.
 */
export function validateSource(source, fileName) {
  const problems = [];
  if (!FILE_NAME_RE.test(fileName)) throw new IngestError(`Nombre de archivo inválido: "${fileName}" (use letras, números, punto, guion; extensión .tsp).`);
  const bytes = Buffer.byteLength(source, "utf8");
  if (bytes === 0) throw new IngestError("El archivo está vacío.");
  if (bytes > config.maxSourceBytes) throw new IngestError(`El archivo supera el máximo de ${config.maxSourceBytes} bytes.`);
  if (source.includes("\0")) throw new IngestError("El archivo contiene caracteres nulos.");

  const script = parse(source);
  if (script.parseDiagnostics.length) {
    throw new IngestError("Error de sintaxis en el archivo TypeSpec.", script.parseDiagnostics.map((d) => ({ code: d.code, message: d.message })));
  }

  const imports = [];
  let namespace = null;
  for (const s of script.statements) {
    if (s.kind === SyntaxKind.ImportStatement) {
      const p = s.path.value;
      if (p.startsWith("./") || p.startsWith("../")) {
        if (!p.endsWith(".tsp")) problems.push(`Importación no permitida "${p}": solo se admiten archivos .tsp.`);
        else if (/[\\\0]/.test(p)) problems.push(`Ruta de importación inválida "${p}".`);
        else imports.push(p);
      } else if (!ALLOWED_PACKAGES.has(p)) {
        problems.push(`Importación no permitida "${p}". Permitidas: ${[...ALLOWED_PACKAGES].join(", ")} y rutas relativas a archivos .tsp.`);
      }
    } else if (s.kind === SyntaxKind.DecoratorDeclarationStatement || s.kind === SyntaxKind.FunctionDeclarationStatement) {
      problems.push("No se permiten declaraciones 'extern' (decoradores o funciones) en esquemas del repositorio.");
    } else if (s.kind === SyntaxKind.NamespaceStatement && !namespace) {
      namespace = namespaceName(s);
    }
  }
  if (!namespace) problems.push("El archivo debe declarar un namespace (por ejemplo: namespace Chile.Comunes;).");
  else if (!/^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$/.test(namespace)) problems.push(`Namespace inválido: "${namespace}".`);
  if (problems.length) throw new IngestError("El archivo no cumple las reglas del repositorio.", problems.map((m) => ({ message: m })));

  return { namespace, dir: namespaceToDir(namespace), imports };
}

/** Resuelve las importaciones relativas contra la ruta del archivo (dentro del repositorio). */
export function resolveImports(filePath, imports) {
  const base = posix.dirname(filePath);
  const out = [];
  for (const imp of imports) {
    const r = posix.normalize(posix.join(base, imp));
    if (r.startsWith("..") || posix.isAbsolute(r)) throw new IngestError(`La importación "${imp}" sale del repositorio.`);
    out.push(r);
  }
  return out;
}
