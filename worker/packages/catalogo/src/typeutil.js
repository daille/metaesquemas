import { getNamespaceFullName, isTemplateDeclaration } from "@typespec/compiler";

const BUILTIN_ROOT = "TypeSpec";
const SKIP_NS = new Set(["TypeSpec", "Catalogo"]);

export const nsName = (ns) => (ns ? getNamespaceFullName(ns) : "");

/** ¿Es un tipo declarado por el usuario (no de la librería estándar ni del catálogo)? */
export function isUserType(t) {
  if (!t || !("namespace" in t) || !t.namespace) return false;
  const root = nsName(t.namespace).split(".")[0];
  if (root === "" || SKIP_NS.has(root) || root.startsWith("@")) return false;
  return true;
}

export const isBuiltinScalar = (t) => t?.kind === "Scalar" && !isUserType(t);

export const isArrayType = (t) => t?.kind === "Model" && t.name === "Array" && !isUserType(t) && !!t.indexer;
export const isRecordType = (t) => t?.kind === "Model" && t.name === "Record" && !isUserType(t) && !!t.indexer;
export const isTemplateInstance = (t) => t?.kind === "Model" && !!t.templateMapper && !isTemplateDeclaration(t) && isUserType(t) && !!t.name;

/** Nombre simple legible del tipo (sin colisiones entre namespaces dentro de un mismo archivo generado). */
export function simpleName(t) {
  if (isTemplateInstance(t)) {
    const args = (t.templateMapper?.args || []).map((a) => (a.entityKind === "Type" ? simpleName(a) : String(a?.value ?? "v")));
    return `${t.name}_${args.join("_")}`.replace(/[^A-Za-z0-9_]/g, "_");
  }
  return t.name || "Anonimo";
}

export const fullName = (t) => {
  const ns = nsName(t.namespace);
  const n = simpleName(t);
  return ns ? `${ns}.${n}` : n;
};

/** Texto de un tipo para mostrar en el catálogo. */
export function display(t) {
  if (!t) return "unknown";
  switch (t.kind) {
    case "Scalar": return isUserType(t) ? fullName(t) : t.name;
    case "Enum": case "Interface": return fullName(t);
    case "Union": return t.name ? fullName(t) : [...t.variants.values()].map((v) => display(v.type)).join(" | ");
    case "UnionVariant": return display(t.type);
    case "Model":
      if (isArrayType(t)) return `${display(t.indexer.value)}[]`;
      if (isRecordType(t)) return `Record<${display(t.indexer.value)}>`;
      return t.name ? fullName(t) : "{ … }";
    case "String": return JSON.stringify(t.value);
    case "Number": case "Boolean": return String(t.value);
    case "Tuple": return `[${t.values.map(display).join(", ")}]`;
    case "Intrinsic": return t.name;
    default: return t.kind;
  }
}

/** Nombre del escalar estándar al que se reduce (sigue `extends` de escalares de usuario). */
export function builtinRoot(t) {
  let cur = t;
  while (cur && cur.kind === "Scalar" && isUserType(cur)) cur = cur.baseScalar;
  return cur?.kind === "Scalar" ? cur.name : "unknown";
}

/**
 * Referencias directas a otros tipos de usuario: [{ type, relation }].
 * relation: composicion | extension | variante | uso | retorno
 */
export function refs(t) {
  const out = [];
  const seen = new Set();
  const add = (target, relation) => {
    if (!target) return;
    const key = relation + "|" + (target.kind) + "|" + (isUserType(target) ? fullName(target) : display(target));
    if (seen.has(key)) return;
    seen.add(key);
    out.push({ type: target, relation });
  };
  const visit = (x, relation, depth = 0) => {
    if (!x || depth > 8) return;
    switch (x.kind) {
      case "Model":
        if (isArrayType(x) || isRecordType(x)) return visit(x.indexer.value, relation, depth + 1);
        if (x.name && isUserType(x)) return add(x, relation);
        if (!x.name) for (const p of x.properties.values()) visit(p.type, relation, depth + 1); // modelo anónimo / intersección
        return;
      case "Scalar": case "Enum": return isUserType(x) ? add(x, relation) : undefined;
      case "Union":
        if (x.name && isUserType(x)) return add(x, relation);
        for (const v of x.variants.values()) visit(v.type, relation, depth + 1);
        return;
      case "UnionVariant": return visit(x.type, relation, depth + 1);
      case "Tuple": for (const v of x.values) visit(v, relation, depth + 1); return;
      default: return;
    }
  };

  switch (t.kind) {
    case "Model":
      if (t.baseModel && isUserType(t.baseModel)) add(t.baseModel, "extension");
      if (t.sourceModel && isUserType(t.sourceModel) && t.sourceModel !== t.baseModel) add(t.sourceModel, "extension");
      for (const p of t.properties.values()) visit(p.type, "composicion");
      if (t.indexer) visit(t.indexer.value, "composicion");
      break;
    case "Scalar":
      if (t.baseScalar && isUserType(t.baseScalar)) add(t.baseScalar, "extension");
      break;
    case "Union":
      for (const v of t.variants.values()) visit(v.type, "variante");
      break;
    case "Interface":
      for (const op of t.operations.values()) {
        for (const p of op.parameters.properties.values()) visit(p.type, "uso");
        visit(op.returnType, "retorno");
      }
      break;
    default: break;
  }
  return out;
}

/** Cierre transitivo (dependencias primero) de los tipos de usuario alcanzables desde `roots`. */
export function closure(roots) {
  const order = [];
  const state = new Map();
  const walk = (t) => {
    const key = fullName(t);
    if (state.get(key)) return;
    state.set(key, 1);
    for (const r of refs(t)) walk(r.type);
    order.push(t);
  };
  roots.forEach(walk);
  return order;
}

export const xmlEsc = (s) => String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
