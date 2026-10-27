import { test, before } from "node:test";
import assert from "node:assert/strict";
import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";

// El almacén es temporal: la configuración se lee al importar los módulos.
process.env.DATA_DIR = await mkdtemp(join(tmpdir(), "metaesquemas-test-"));
const { ingestSource, IngestError } = await import("../src/ingest.js");
const { validateSource } = await import("../src/validate.js");
const store = await import("../src/store.js");

const read = (p) => readFile(new URL(p, import.meta.url), "utf8");
const sample = (n) => read(`../../samples/chile/${n}`);
const rejects = (p, re) => assert.rejects(p, (e) => e instanceof IngestError && re.test(e.message + JSON.stringify(e.details)));

const IMP = 'import "@catalogo/typespec";\n';
const USING = "using Catalogo;\n";
const HEAD = IMP + USING;

test("validación: rechaza importar JavaScript, paquetes no permitidos y 'extern'", () => {
  const ns = "namespace A.B;\n";
  const bad = (src, name, re) => assert.throws(() => validateSource(src, name), (e) => e instanceof IngestError && re.test(e.message + JSON.stringify(e.details)), src.slice(0, 60));
  bad(IMP + 'import "./x.js";\n' + USING + ns, "a.tsp", /solo se admiten archivos \.tsp/);
  bad('import "evil-package";\n' + ns, "a.tsp", /no permitida/);
  bad('import "@typespec/compiler";\n' + ns, "a.tsp", /no permitida/);
  bad(HEAD + ns + "extern dec malo(t: unknown);\n", "a.tsp", /extern/);
  bad(HEAD + "model M {}", "a.tsp", /debe declarar un namespace/);
  bad(HEAD + ns + "model M {", "a.tsp", /sintaxis/);
  for (const name of ["../a.tsp", "a b.tsp", "a.js", ".tsp", "a/b.tsp"]) bad(HEAD + ns, name, /Nombre de archivo inválido/);
  const ok = validateSource(IMP + 'import "../x/y.tsp";\n' + USING + ns + "model M {}", "ok.tsp");
  assert.deepEqual([ok.namespace, ok.dir, ok.imports], ["A.B", "a/b", ["../x/y.tsp"]]);
});

test("ingesta: publica, extrae metadatos y calcula 'usado por'", async () => {
  const r1 = await ingestSource({ source: await sample("basicos/tipos.tsp"), fileName: "tipos.tsp" });
  assert.equal(r1.status, "published");
  assert.equal(r1.rel, "chile/basicos/tipos.tsp");
  await ingestSource({ source: await sample("comunes/personas.tsp"), fileName: "personas.tsp" });
  await ingestSource({ source: await sample("documentos/certificados.tsp"), fileName: "certificados.tsp" });
  await ingestSource({ source: await sample("servicios/personas/personas-api.tsp"), fileName: "personas-api.tsp" });

  const idx = await store.readIndex();
  const by = Object.fromEntries(idx.entries.map((e) => [e.id, e]));
  assert.equal(idx.entries.length, 16);
  assert.equal(by["Chile.Basicos.Rut"].tipo, "basico");
  assert.equal(by["Chile.Comunes.Persona"].tipo, "estructurado");
  assert.equal(by["Chile.Documentos.CertificadoNacimiento"].tipo, "documental");
  assert.equal(by["Chile.Servicios.Personas.Personas"].tipo, "servicio");
  // responsable declarado, heredado del namespace y sobrescrito en el tipo
  assert.equal(by["Chile.Comunes.Direccion"].responsable.nombre, "División de Datos e Interoperabilidad");
  assert.equal(by["Chile.Comunes.Direccion"].responsable.heredado, true);
  assert.equal(by["Chile.Comunes.ContribuyenteLegado"].responsable.heredado, false);
  assert.deepEqual(by["Chile.Comunes.ContribuyenteLegado"].vigencia, { desde: "2015-01-01", hasta: "2023-12-31", heredada: false });
  // dependencias y uso inverso
  const usedBy = (id) => by[id].usedBy.map((u) => u.id).sort();
  assert.deepEqual(usedBy("Chile.Basicos.Rut"), ["Chile.Comunes.ContribuyenteLegado", "Chile.Comunes.Persona", "Chile.Servicios.Personas.Certificados", "Chile.Servicios.Personas.Personas"]);
  assert.ok(usedBy("Chile.Comunes.Persona").includes("Chile.Documentos.CertificadoNacimiento"));
  assert.ok(by["Chile.Comunes.PersonaResidente"].dependencies.some((d) => d.id === "Chile.Comunes.Persona" && d.relation === "extension"));
  assert.ok(by["Chile.Servicios.Personas.Personas"].dependencies.some((d) => d.id === "Chile.Comunes.Pagina"));
  // las plantillas no se proyectan; los servicios tienen OpenAPI
  assert.deepEqual(by["Chile.Comunes.Pagina"].projections, ["tsp"]);
  assert.ok(by["Chile.Servicios.Personas.Personas"].projections.includes("openapi"));
  assert.ok(by["Chile.Comunes.Persona"].projections.includes("jsonschema"));
});

test("proyecciones: JSON Schema autocontenido, XSD y Protobuf", async () => {
  const dir = store.versionDir("chile/comunes/personas.tsp", 1) + "/artifacts";
  const js = JSON.parse(await readFile(`${dir}/jsonschema/Chile.Comunes.Persona.json`, "utf8"));
  assert.equal(js.properties.rut.$ref, "#/$defs/Rut");
  assert.ok(js.$defs.Rut.pattern, "Rut debe traer su patrón");
  assert.ok(js.$defs.Direccion && js.$defs.EstadoCivil, "dependencias insertadas en $defs");
  assert.ok(!JSON.stringify(js).includes('.json"'), "sin referencias a archivos externos");
  const xsd = await readFile(`${dir}/xsd/Chile.Comunes.Persona.xsd`, "utf8");
  assert.match(xsd, /<xs:complexType name="Persona">/);
  assert.match(xsd, /<xs:pattern value="\\d\{7,8\}-\[\\dkK\]"\/>/); // sin ^ ni $
  const proto = await readFile(`${dir}/proto/Chile.Comunes.Persona.proto`, "utf8");
  assert.match(proto, /syntax = "proto3";/);
  assert.match(proto, /optional EstadoCivil estadoCivil = 4;/);
});

test("versionado: sin cambios no crea versión; un cambio crea v2 y recompila dependientes", async () => {
  const src = await sample("comunes/personas.tsp");
  const same = await ingestSource({ source: src, fileName: "personas.tsp" });
  assert.equal(same.status, "unchanged");
  assert.equal(await store.readCurrent("chile/comunes/personas.tsp"), 1);

  const changed = src.replace("numero?: string;", "numero?: string;\n  departamento?: string;");
  const r = await ingestSource({ source: changed, fileName: "personas.tsp" });
  assert.equal(r.status, "published");
  assert.equal(r.version, 2);
  assert.deepEqual(r.dependents.map((d) => d.file).sort(), ["chile/documentos/certificados.tsp", "chile/servicios/personas/personas-api.tsp"]);
  assert.ok(r.dependents.every((d) => d.status === "published" && d.version === 2));

  const meta = await store.readMeta("chile/documentos/certificados.tsp", 2);
  assert.match(meta.file.motivo, /dependencia actualizada: chile\/comunes\/personas\.tsp/);
  const proto = await readFile(store.versionDir("chile/documentos/certificados.tsp", 2) + "/artifacts/proto/Chile.Documentos.CertificadoNacimiento.proto", "utf8");
  assert.match(proto, /departamento/, "la proyección del dependiente refleja el cambio");
  // la versión 1 sigue disponible, intacta
  assert.doesNotMatch(await readFile(store.versionDir("chile/comunes/personas.tsp", 1) + "/source.tsp", "utf8"), /departamento/);
  assert.equal((await store.readMeta("chile/comunes/personas.tsp", 2)).file.history.length, 2);
});

test("un cambio que rompe a un dependiente se informa y conserva su versión anterior", async () => {
  const tipos = (await sample("basicos/tipos.tsp")).replace(/scalar Rut extends string;/, "scalar RutNuevo extends string;");
  const r = await ingestSource({ source: tipos, fileName: "tipos.tsp" });
  assert.equal(r.status, "published");
  const broken = r.dependents.filter((d) => d.status === "error");
  assert.ok(broken.length >= 1, "debe informar dependientes rotos");
  assert.equal(await store.readCurrent("chile/documentos/certificados.tsp"), 2, "conserva su versión anterior");
});

test("rechazos: sin responsable, error de compilación, nombre duplicado, importación inexistente", async () => {
  await rejects(ingestSource({ source: HEAD + "namespace Otro.Area;\nmodel Huerfano { x: string; }", fileName: "huerfano.tsp" }), /responsable/i);
  await rejects(ingestSource({ source: HEAD + '@responsable("X")\nnamespace Otro.Area;\nmodel Malo { x: NoExiste; }', fileName: "malo.tsp" }), /compilación/i);
  await rejects(ingestSource({ source: IMP + 'import "../nada/no-existe.tsp";\n' + USING + '@responsable("X")\nnamespace Otro.Area;\nmodel M {}', fileName: "m.tsp" }), /no existe en el repositorio/);
  await rejects(ingestSource({ source: HEAD + '@responsable("X")\nnamespace Chile.Comunes;\nmodel Direccion { otra: string; }', fileName: "duplicado.tsp" }), /ya está definido/i);
  await rejects(ingestSource({ source: HEAD + '@responsable("X")\n@vigencia("2025-13-45")\nnamespace Otro.Area;\nmodel V {}', fileName: "v.tsp" }), /fecha/i);
  await rejects(ingestSource({ source: HEAD + '@responsable("X")\n@vigencia("2025-06-01", "2025-01-01")\nnamespace Otro.Area;\nmodel V {}', fileName: "v2.tsp" }), /desde/);
  // nada de lo rechazado quedó publicado
  const idx = await store.readIndex();
  assert.ok(!idx.entries.some((e) => /Otro\.Area/.test(e.id)));
});
