import * as tsp from "@typespec/compiler";
import { keys } from "./lib.js";
import { isUserType, isTemplateInstance, isArrayType, isRecordType, nsName, fullName, display, builtinRoot, refs } from "./typeutil.js";

const { getDoc, getSourceLocation, isTemplateDeclaration, getService } = tsp;

function* walk(ns) {
  yield ns;
  for (const c of ns.namespaces.values()) yield* walk(c);
}

/** Busca un valor de estado en el tipo y, si falta, en sus namespaces contenedores (herencia). */
function pick(program, key, t) {
  const map = program.stateMap(key);
  let cur = t, first = true;
  while (cur) {
    const v = map.get(cur);
    if (v) return { value: v, inherited: !first, from: cur.kind === "Namespace" ? nsName(cur) : fullName(cur) };
    cur = cur.namespace;
    first = false;
  }
  return null;
}

function lineOf(file, pos) {
  const before = file.text.slice(0, pos);
  return before.split("\n").length;
}

/** Texto fuente de la declaración, incluyendo el comentario de documentación contiguo. */
function snippet(t) {
  const loc = getSourceLocation(t);
  if (!loc || !loc.file) return null;
  const { text } = loc.file;
  let start = loc.pos;
  const window = text.slice(Math.max(0, start - 3000), start);
  const m = /(?:\/\*[\s\S]*?\*\/[ \t]*\r?\n[ \t]*|(?:[ \t]*\/\/[^\n]*\n)+[ \t]*)$/.exec(window);
  if (m) start -= m[0].length;
  return { text: text.slice(start, loc.end), startLine: lineOf(loc.file, start), endLine: lineOf(loc.file, loc.end), file: loc.file.path };
}

function constraints(program, t) {
  const c = {};
  const set = (k, v) => { if (v !== undefined) c[k] = v; };
  set("minLength", tsp.getMinLength(program, t)); set("maxLength", tsp.getMaxLength(program, t));
  set("pattern", tsp.getPattern(program, t));
  set("minValue", tsp.getMinValue(program, t)); set("maxValue", tsp.getMaxValue(program, t));
  return Object.keys(c).length ? c : undefined;
}

function describeMembers(program, t) {
  switch (t.kind) {
    case "Model": {
      const members = [...t.properties.values()].map((p) => ({
        name: p.name, type: display(p.type), optional: !!p.optional, doc: getDoc(program, p) || undefined, constraints: constraints(program, p),
      }));
      return { members, extends: t.baseModel && t.baseModel.name ? display(t.baseModel) : undefined };
    }
    case "Scalar": return { baseType: t.baseScalar ? display(t.baseScalar) : undefined, builtin: builtinRoot(t), constraints: constraints(program, t) };
    case "Enum": return { values: [...t.members.values()].map((m) => ({ name: m.name, value: m.value ?? m.name, doc: getDoc(program, m) || undefined })) };
    case "Union": return { variants: [...t.variants.values()].map((v) => ({ name: typeof v.name === "string" ? v.name : undefined, type: display(v.type) })) };
    case "Interface": return {
      operations: [...t.operations.values()].map((op) => ({
        name: op.name, doc: getDoc(program, op) || undefined,
        params: [...op.parameters.properties.values()].map((p) => ({ name: p.name, type: display(p.type), optional: !!p.optional })),
        returns: display(op.returnType),
      })),
    };
    default: return {};
  }
}

/** Dependencias directas como [{id, relation}], expandiendo instancias de plantillas. */
function dependencies(t) {
  const out = [];
  const seen = new Set();
  const selfId = fullName(t);
  const addId = (id, relation) => {
    const k = id + "|" + relation;
    if (id === selfId || seen.has(k)) return;
    seen.add(k);
    out.push({ id, relation });
  };
  const visit = (type, relation, depth) => {
    if (depth > 6) return;
    if (isTemplateInstance(type)) {
      const decl = type.namespace?.models.get(type.name);
      if (decl) addId(fullName(decl), relation);
      for (const r of refs(type)) visit(r.type, r.relation === "extension" ? relation : r.relation, depth + 1);
    } else addId(fullName(type), relation);
  };
  for (const r of refs(t)) visit(r.type, r.relation, 0);
  return out;
}

export function collectEntries(program, mainFile) {
  const entries = [];
  const global = program.getGlobalNamespaceType();
  for (const ns of walk(global)) {
    const root = nsName(ns).split(".")[0];
    if (!root || root === "TypeSpec" || root === "Catalogo" || root.startsWith("@")) continue;
    const groups = [ns.models, ns.scalars, ns.enums, ns.unions, ns.interfaces];
    for (const g of groups) {
      for (const t of g.values()) {
        if (!t.name || !isUserType(t) || isTemplateInstance(t)) continue;
        if ((t.kind === "Model" && (isArrayType(t) || isRecordType(t)))) continue;
        const loc = getSourceLocation(t);
        if (!loc?.file || loc.file.path !== mainFile) continue;

        const resp = pick(program, keys.responsable, t);
        const vig = pick(program, keys.vigencia, t);
        const tipoExplicit = program.stateMap(keys.tipo).get(t);
        const tipo = tipoExplicit || (t.kind === "Interface" ? "servicio" : t.kind === "Scalar" || t.kind === "Enum" ? "basico" : "estructurado");
        const svc = t.kind === "Interface" && t.namespace ? getService(program, t.namespace) : undefined;
        const deprecated = typeof tsp.getDeprecationDetails === "function" ? tsp.getDeprecationDetails(program, t) : undefined;

        entries.push({
          type: t, // se retira antes de serializar
          id: fullName(t), name: t.name, namespace: nsName(t.namespace), kind: t.kind.toLowerCase(),
          tipo, tipoInferido: !tipoExplicit,
          plantilla: isTemplateDeclaration(t) ? { parametros: (t.node?.templateParameters || []).map((p) => p.id.sv) } : undefined,
          doc: getDoc(program, t) || undefined,
          responsable: resp ? { ...resp.value, heredado: resp.inherited, origen: resp.inherited ? resp.from : undefined } : null,
          vigencia: vig ? { ...vig.value, heredada: vig.inherited } : null,
          deprecado: deprecated ? { mensaje: deprecated.message } : undefined,
          servicio: svc ? { titulo: svc.title } : undefined,
          ...describeMembers(program, t),
          dependencies: dependencies(t),
          source: snippet(t),
        });
      }
    }
  }
  return entries.sort((a, b) => a.id.localeCompare(b.id));
}
