import { getDoc, getMinLength, getMaxLength, getPattern, getMinValue, getMaxValue } from "@typespec/compiler";
import { isUserType, isArrayType, isRecordType, isTemplateInstance, simpleName, fullName, nsName, builtinRoot, closure, xmlEsc, refs } from "./typeutil.js";

const XS = {
  string: "xs:string", url: "xs:anyURI", boolean: "xs:boolean", bytes: "xs:base64Binary",
  int8: "xs:byte", int16: "xs:short", int32: "xs:int", int64: "xs:long", safeint: "xs:long", integer: "xs:integer",
  uint8: "xs:unsignedByte", uint16: "xs:unsignedShort", uint32: "xs:unsignedInt", uint64: "xs:unsignedLong",
  float32: "xs:float", float64: "xs:double", float: "xs:double", numeric: "xs:decimal", decimal: "xs:decimal", decimal128: "xs:decimal",
  utcDateTime: "xs:dateTime", offsetDateTime: "xs:dateTime", plainDate: "xs:date", plainTime: "xs:time", duration: "xs:duration",
  unknown: "xs:anyType",
};
const NUMERIC = new Set(["int8", "int16", "int32", "int64", "safeint", "integer", "uint8", "uint16", "uint32", "uint64", "float32", "float64", "float", "numeric", "decimal", "decimal128"]);

/** XSD no admite ^ ni $ (la expresión está implícitamente anclada). */
const xsdPattern = (p) => p.replace(/^\^/, "").replace(/\$$/, "");

function facets(program, target, root) {
  const out = [];
  const num = NUMERIC.has(root);
  const minL = getMinLength(program, target), maxL = getMaxLength(program, target), pat = getPattern(program, target);
  const minV = getMinValue(program, target), maxV = getMaxValue(program, target);
  if (!num) {
    if (minL !== undefined) out.push(`<xs:minLength value="${minL}"/>`);
    if (maxL !== undefined) out.push(`<xs:maxLength value="${maxL}"/>`);
    if (pat !== undefined) out.push(`<xs:pattern value="${xmlEsc(xsdPattern(pat))}"/>`);
  } else {
    if (minV !== undefined) out.push(`<xs:minInclusive value="${minV}"/>`);
    if (maxV !== undefined) out.push(`<xs:maxInclusive value="${maxV}"/>`);
  }
  return out;
}

export function generateXsd(program, rootEntryType) {
  const roots = rootEntryType.kind === "Interface" ? refs(rootEntryType).map((r) => r.type) : [rootEntryType];
  const types = closure(roots);
  if (types.length === 0) return null;

  // Nombres: simple salvo colisión entre namespaces distintos.
  const byName = new Map();
  for (const t of types) byName.set(simpleName(t), (byName.get(simpleName(t)) || new Set()).add(fullName(t)));
  const nameOf = (t) => (byName.get(simpleName(t)).size > 1 ? fullName(t).replace(/\./g, "_") : simpleName(t));

  const targetNs = "urn:catalogo:" + (nsName(rootEntryType.namespace) || "default").toLowerCase();
  const lines = [];
  const doc = (t, ind) => {
    const d = getDoc(program, t);
    return d ? `${ind}<xs:annotation><xs:documentation xml:lang="es">${xmlEsc(d)}</xs:documentation></xs:annotation>\n` : "";
  };

  /** Devuelve { attrs, inline } para usar como tipo de un elemento. */
  function typeOf(t, prop, ind) {
    if (!t) return { attrs: 'type="xs:anyType"' };
    switch (t.kind) {
      case "Scalar": {
        if (!isUserType(t)) {
          const root = t.name;
          const f = prop ? facets(program, prop, root) : [];
          if (f.length) return { inline: `${ind}<xs:simpleType><xs:restriction base="${XS[root] || "xs:string"}">${f.join("")}</xs:restriction></xs:simpleType>\n` };
          return { attrs: `type="${XS[root] || "xs:string"}"` };
        }
        const f = prop ? facets(program, prop, builtinRoot(t)) : [];
        if (f.length) return { inline: `${ind}<xs:simpleType><xs:restriction base="tns:${nameOf(t)}">${f.join("")}</xs:restriction></xs:simpleType>\n` };
        return { attrs: `type="tns:${nameOf(t)}"` };
      }
      case "Enum": return { attrs: `type="tns:${nameOf(t)}"` };
      case "Model":
        if (isArrayType(t)) return typeOf(t.indexer.value, prop, ind);
        if (isRecordType(t)) return { inline: `${ind}<xs:complexType><xs:sequence><xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType>\n` };
        if (t.name) return { attrs: `type="tns:${nameOf(t)}"` };
        return { inline: `${ind}<xs:complexType>\n${modelBody(t, ind + "  ")}${ind}</xs:complexType>\n` };
      case "Union": {
        if (t.name && isUserType(t)) return { attrs: `type="tns:${nameOf(t)}"` };
        const vs = [...t.variants.values()].map((v) => v.type).filter((x) => x.kind !== "Intrinsic");
        if (vs.length === 1) return typeOf(vs[0], prop, ind);
        if (vs.length && vs.every((x) => x.kind === "String")) return { inline: `${ind}<xs:simpleType><xs:restriction base="xs:string">${vs.map((x) => `<xs:enumeration value="${xmlEsc(x.value)}"/>`).join("")}</xs:restriction></xs:simpleType>\n` };
        return { attrs: 'type="xs:anyType"', comment: "unión no representable en XSD" };
      }
      case "String": return { attrs: 'type="xs:string"' };
      case "Number": return { attrs: 'type="xs:decimal"' };
      case "Boolean": return { attrs: 'type="xs:boolean"' };
      default: return { attrs: 'type="xs:anyType"' };
    }
  }

  const isNullable = (t) => t?.kind === "Union" && !t.name && [...t.variants.values()].some((v) => v.type.kind === "Intrinsic" && v.type.name === "null");

  function element(prop, ind) {
    let t = prop.type;
    const many = isArrayType(t);
    const typeForAttrs = many ? t.indexer.value : t;
    const { attrs, inline, comment } = typeOf(typeForAttrs, prop, ind + "  ");
    const occurs = [`minOccurs="${prop.optional ? 0 : 1}"`];
    if (many) occurs.push('maxOccurs="unbounded"');
    if (isNullable(typeForAttrs)) occurs.push('nillable="true"');
    const head = `${ind}<xs:element name="${xmlEsc(prop.name)}" ${inline ? "" : attrs + " "}${occurs.join(" ")}`;
    const d = getDoc(program, prop);
    const body = (d ? `${ind}  <xs:annotation><xs:documentation xml:lang="es">${xmlEsc(d)}</xs:documentation></xs:annotation>\n` : "") + (inline || "");
    return (comment ? `${ind}<!-- ${comment} -->\n` : "") + (body ? `${head}>\n${body}${ind}</xs:element>\n` : `${head}/>\n`);
  }

  function modelBody(m, ind) {
    const props = [...m.properties.values()].filter((p) => !(m.baseModel && m.baseModel.properties.has(p.name) && !m.sourceModel));
    const seq = props.map((p) => element(p, ind + "  ")).join("");
    return seq ? `${ind}<xs:sequence>\n${seq}${ind}</xs:sequence>\n` : `${ind}<xs:sequence/>\n`;
  }

  for (const t of types) {
    const n = nameOf(t);
    switch (t.kind) {
      case "Scalar": {
        const root = builtinRoot(t);
        const base = isUserType(t.baseScalar) ? `tns:${nameOf(t.baseScalar)}` : XS[t.baseScalar?.name] || XS[root] || "xs:string";
        const f = facets(program, t, root);
        lines.push(`  <xs:simpleType name="${n}">\n${doc(t, "    ")}    <xs:restriction base="${base}">${f.join("")}</xs:restriction>\n  </xs:simpleType>\n`);
        break;
      }
      case "Enum": {
        const vals = [...t.members.values()].map((m) => `      <xs:enumeration value="${xmlEsc(m.value ?? m.name)}"/>\n`).join("");
        lines.push(`  <xs:simpleType name="${n}">\n${doc(t, "    ")}    <xs:restriction base="xs:string">\n${vals}    </xs:restriction>\n  </xs:simpleType>\n`);
        break;
      }
      case "Union": {
        const vs = [...t.variants.values()];
        if (vs.every((v) => v.type.kind === "String")) {
          lines.push(`  <xs:simpleType name="${n}">\n${doc(t, "    ")}    <xs:restriction base="xs:string">\n${vs.map((v) => `      <xs:enumeration value="${xmlEsc(v.type.value)}"/>\n`).join("")}    </xs:restriction>\n  </xs:simpleType>\n`);
        } else {
          const els = vs.map((v) => {
            const { attrs, inline } = typeOf(v.type, null, "        ");
            const nm = typeof v.name === "string" ? v.name : display0(v.type);
            return inline ? `      <xs:element name="${xmlEsc(nm)}">\n${inline}      </xs:element>\n` : `      <xs:element name="${xmlEsc(nm)}" ${attrs}/>\n`;
          }).join("");
          lines.push(`  <xs:complexType name="${n}">\n${doc(t, "    ")}    <xs:choice>\n${els}    </xs:choice>\n  </xs:complexType>\n`);
        }
        break;
      }
      case "Model": {
        if (t.baseModel && isUserType(t.baseModel) && !t.sourceModel) {
          lines.push(`  <xs:complexType name="${n}">\n${doc(t, "    ")}    <xs:complexContent>\n      <xs:extension base="tns:${nameOf(t.baseModel)}">\n${modelBody(t, "        ")}      </xs:extension>\n    </xs:complexContent>\n  </xs:complexType>\n`);
        } else {
          lines.push(`  <xs:complexType name="${n}">\n${doc(t, "    ")}${modelBody(t, "    ")}  </xs:complexType>\n`);
        }
        break;
      }
      default: break;
    }
  }
  // Elemento raíz para los modelos del propio esquema.
  for (const t of roots) if (t.kind === "Model" && !isRecordType(t) && !isArrayType(t)) lines.push(`  <xs:element name="${nameOf(t)}" type="tns:${nameOf(t)}"/>\n`);

  return `<?xml version="1.0" encoding="UTF-8"?>
<!-- Generado automáticamente desde TypeSpec (${fullName(rootEntryType)}). No edite: se recalcula al cargar el archivo fuente. -->
<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:tns="${targetNs}" targetNamespace="${targetNs}" elementFormDefault="qualified">
${lines.join("")}</xs:schema>
`;
}

function display0(t) { return (t.name || t.kind || "variante").toString().replace(/[^A-Za-z0-9_]/g, "_"); }
