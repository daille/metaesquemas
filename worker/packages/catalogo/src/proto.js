import { getDoc, isErrorModel } from "@typespec/compiler";
import { isUserType, isArrayType, isRecordType, simpleName, fullName, nsName, closure, refs } from "./typeutil.js";

const SCALAR = {
  string: "string", url: "string", boolean: "bool", bytes: "bytes",
  int8: "int32", int16: "int32", int32: "int32", int64: "int64", safeint: "int64", integer: "int64",
  uint8: "uint32", uint16: "uint32", uint32: "uint32", uint64: "uint64",
  float32: "float", float64: "double", float: "double",
  numeric: "string", decimal: "string", decimal128: "string",   // sin pérdida de precisión
  plainDate: "string", plainTime: "string",
};
const WKT = { utcDateTime: ["google.protobuf.Timestamp", "google/protobuf/timestamp.proto"], offsetDateTime: ["google.protobuf.Timestamp", "google/protobuf/timestamp.proto"],
  duration: ["google.protobuf.Duration", "google/protobuf/duration.proto"], unknown: ["google.protobuf.Any", "google/protobuf/any.proto"] };

const ident = (s) => String(s).replace(/[^A-Za-z0-9_]/g, "_").replace(/^(\d)/, "_$1");
const upperSnake = (s) => s.replace(/([a-z0-9])([A-Z])/g, "$1_$2").replace(/[^A-Za-z0-9]+/g, "_").toUpperCase();
const comment = (text, ind = "") => (text ? text.split("\n").map((l) => `${ind}// ${l}`.trimEnd()).join("\n") + "\n" : "");

export function generateProto(program, rootEntryType) {
  const isService = rootEntryType.kind === "Interface";
  const roots = isService ? refs(rootEntryType).map((r) => r.type) : [rootEntryType];
  const types = closure(roots);
  if (types.length === 0 && !isService) return null;

  const byName = new Map();
  for (const t of types) byName.set(simpleName(t), (byName.get(simpleName(t)) || new Set()).add(fullName(t)));
  const nameOf = (t) => ident(byName.get(simpleName(t))?.size > 1 ? fullName(t) : simpleName(t));

  const imports = new Set();
  const out = [];
  const extraMessages = [];

  /** Tipo proto de `t` (sin repeated/map). Devuelve { type, note?, repeated?, map? }. */
  function protoOf(t) {
    if (!t) return { type: "google.protobuf.Any", imp: "google/protobuf/any.proto" };
    switch (t.kind) {
      case "Scalar": {
        if (isUserType(t)) return protoOf(t.baseScalar);        // los escalares de usuario se aplanan a su tipo base
        if (SCALAR[t.name]) return { type: SCALAR[t.name] };
        if (WKT[t.name]) return { type: WKT[t.name][0], imp: WKT[t.name][1] };
        return { type: "string" };
      }
      case "Enum": return { type: nameOf(t) };
      case "Model":
        if (isArrayType(t)) {
          const inner = protoOf(t.indexer.value);
          if (inner.repeated || inner.map) return { type: "google.protobuf.Value", imp: "google/protobuf/struct.proto", note: "arreglo anidado" };
          return { ...inner, repeated: true };
        }
        if (isRecordType(t)) {
          const inner = protoOf(t.indexer.value);
          if (inner.repeated || inner.map) return { type: "google.protobuf.Struct", imp: "google/protobuf/struct.proto", note: "mapa con valores compuestos" };
          return { type: `map<string, ${inner.type}>`, imp: inner.imp, map: true };
        }
        if (t.name) return { type: nameOf(t) };
        return { type: "google.protobuf.Struct", imp: "google/protobuf/struct.proto", note: "modelo anónimo" };
      case "Union": {
        if (t.name && isUserType(t)) return { type: nameOf(t) };
        const vs = [...t.variants.values()].map((v) => v.type).filter((x) => !(x.kind === "Intrinsic" && x.name === "null"));
        if (vs.length === 1) return protoOf(vs[0]);
        if (vs.length && vs.every((x) => x.kind === "String")) return { type: "string" };
        return { type: "google.protobuf.Any", imp: "google/protobuf/any.proto", note: "unión sin nombre" };
      }
      case "String": return { type: "string" };
      case "Number": return { type: Number.isInteger(t.value) ? "int64" : "double" };
      case "Boolean": return { type: "bool" };
      default: return { type: "google.protobuf.Any", imp: "google/protobuf/any.proto" };
    }
  }
  const isNullable = (t) => t?.kind === "Union" && !t.name && [...t.variants.values()].some((v) => v.type.kind === "Intrinsic" && v.type.name === "null");

  function fieldsOf(m) {
    // proto3 no tiene herencia: se aplanan las propiedades heredadas primero.
    const chain = [];
    for (let b = m.baseModel; b && isUserType(b); b = b.baseModel) chain.unshift(b);
    const props = [];
    for (const b of chain) for (const p of b.properties.values()) props.push({ p, from: b });
    for (const p of m.properties.values()) if (!props.some((x) => x.p.name === p.name)) props.push({ p, from: m });
    return props;
  }

  function field(p, n, ind, from) {
    const info = protoOf(p.type);
    if (info.imp) imports.add(info.imp);
    const label = info.repeated ? "repeated " : p.optional || isNullable(p.type) ? (info.map ? "" : "optional ") : "";
    const d = getDoc(program, p);
    const notes = [d, info.note && `Nota: ${info.note}.`].filter(Boolean).join(" ");
    return `${comment(notes, ind)}${ind}${label}${info.type} ${ident(p.name)} = ${n};\n`;
  }

  for (const t of types) {
    const n = nameOf(t);
    const d = getDoc(program, t);
    switch (t.kind) {
      case "Scalar": break;                                 // aplanado en los campos
      case "Enum": {
        const prefix = upperSnake(n);
        const vals = [...t.members.values()].map((m, i) => `  ${prefix}_${upperSnake(m.name)} = ${i + 1};\n`).join("");
        out.push(`${comment(d)}enum ${n} {\n  ${prefix}_UNSPECIFIED = 0;\n${vals}}\n`);
        break;
      }
      case "Union": {
        const vs = [...t.variants.values()];
        if (vs.every((v) => v.type.kind === "String")) {
          const prefix = upperSnake(n);
          out.push(`${comment(d)}enum ${n} {\n  ${prefix}_UNSPECIFIED = 0;\n${vs.map((v, i) => `  ${prefix}_${upperSnake(v.type.value)} = ${i + 1};\n`).join("")}}\n`);
        } else {
          const body = vs.filter((v) => !(v.type.kind === "Intrinsic" && v.type.name === "null")).map((v, i) => {
            const info = protoOf(v.type);
            if (info.imp) imports.add(info.imp);
            const nm = typeof v.name === "string" ? v.name : `opcion${i + 1}`;
            return `    ${info.type} ${ident(nm)} = ${i + 1};\n`;
          }).join("");
          out.push(`${comment(d)}message ${n} {\n  oneof valor {\n${body}  }\n}\n`);
        }
        break;
      }
      case "Model": {
        const props = fieldsOf(t);
        const body = props.map(({ p, from }, i) => field(p, i + 1, "  ", from)).join("");
        const inherited = t.baseModel && isUserType(t.baseModel) ? `// Hereda de ${fullName(t.baseModel)} (campos aplanados).\n` : "";
        out.push(`${comment(d)}${inherited}message ${n} {\n${body}}\n`);
        break;
      }
      default: break;
    }
  }

  // Servicios
  let svc = "";
  if (isService) {
    imports.add("google/protobuf/empty.proto");
    const rpcs = [];
    for (const op of rootEntryType.operations.values()) {
      const base = ident(simpleName(rootEntryType) + op.name[0].toUpperCase() + op.name.slice(1));
      const params = [...op.parameters.properties.values()];
      let req;
      if (params.length === 1 && params[0].type.kind === "Model" && params[0].type.name && isUserType(params[0].type)) req = nameOf(params[0].type);
      else if (params.length === 0) req = "google.protobuf.Empty";
      else {
        req = `${base}Request`;
        extraMessages.push(`message ${req} {\n${params.map((p, i) => field(p, i + 1, "  ")).join("")}}\n`);
      }
      // Respuesta: primer tipo de usuario no-error del retorno
      const variants = op.returnType.kind === "Union" ? [...op.returnType.variants.values()].map((v) => v.type) : [op.returnType];
      const ok = variants.find((v) => !(v.kind === "Model" && isErrorModel(program, v)) && !(v.kind === "Intrinsic"));
      let res = "google.protobuf.Empty";
      if (ok) {
        const info = protoOf(ok);
        if (info.imp) imports.add(info.imp);
        if (!info.repeated && !info.map && isUserType(ok) && ok.kind === "Model") res = nameOf(ok);
        else if (ok.kind === "Model" && ok.name && !isArrayType(ok) && !isRecordType(ok) && isUserType(ok)) res = nameOf(ok);
        else {
          res = `${base}Response`;
          extraMessages.push(`message ${res} {\n  ${info.repeated ? "repeated " : ""}${info.type} valor = 1;\n}\n`);
        }
      }
      rpcs.push(`${comment(getDoc(program, op), "  ")}  rpc ${ident(op.name[0].toUpperCase() + op.name.slice(1))}(${req}) returns (${res});\n`);
    }
    svc = `${comment(getDoc(program, rootEntryType))}service ${ident(simpleName(rootEntryType))} {\n${rpcs.join("")}}\n`;
  }

  if (!out.length && !extraMessages.length && !svc) return null;   // p. ej. un escalar: protobuf no tiene alias de tipos

  const pkg = nsName(rootEntryType.namespace).toLowerCase().replace(/[^a-z0-9.]/g, "_");
  const header = `// Generado automáticamente desde TypeSpec (${fullName(rootEntryType)}). No edite: se recalcula al cargar el archivo fuente.
// Los números de campo se asignan por orden de declaración; no reordene ni elimine campos en el .tsp sin versionar el contrato.
syntax = "proto3";

package ${pkg};

${[...imports].sort().map((i) => `import "${i}";`).join("\n")}${imports.size ? "\n" : ""}`;
  return `${header}\n${[...out, ...extraMessages, svc].filter(Boolean).join("\n")}`;
}
