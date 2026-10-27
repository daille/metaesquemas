import { reportDiagnostic, keys } from "./lib.js";

const FECHA = /^\d{4}-\d{2}-\d{2}$/;
const fechaValida = (s) => FECHA.test(s) && !Number.isNaN(Date.parse(s + "T00:00:00Z")) && new Date(s + "T00:00:00Z").toISOString().startsWith(s);

function responsableImpl(context, target, nombre, contacto) {
  if (!nombre || !nombre.trim()) {
    reportDiagnostic(context.program, { code: "responsable-vacio", target });
    return;
  }
  context.program.stateMap(keys.responsable).set(target, { nombre: nombre.trim(), contacto: contacto?.trim() || undefined });
}

function tipoImpl(context, target, valor) {
  const nombre = valor?.name;
  if (!["basico", "estructurado", "documental", "servicio"].includes(nombre)) {
    reportDiagnostic(context.program, { code: "tipo-invalido", format: { valor: String(nombre) }, target });
    return;
  }
  context.program.stateMap(keys.tipo).set(target, nombre);
}

function vigenciaImpl(context, target, desde, hasta) {
  for (const v of [desde, hasta]) {
    if (v !== undefined && !fechaValida(v)) {
      reportDiagnostic(context.program, { code: "fecha-invalida", format: { valor: v }, target });
      return;
    }
  }
  if (hasta && desde > hasta) {
    reportDiagnostic(context.program, { code: "rango-invalido", target });
    return;
  }
  context.program.stateMap(keys.vigencia).set(target, { desde, hasta: hasta || null });
}

/** Registro explícito bajo el namespace `Catalogo` (evita registrar los decoradores como globales). */
export const $decorators = {
  Catalogo: { responsable: responsableImpl, tipo: tipoImpl, vigencia: vigenciaImpl },
};
