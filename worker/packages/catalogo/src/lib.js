import { createTypeSpecLibrary, paramMessage } from "@typespec/compiler";

export const $lib = createTypeSpecLibrary({
  name: "@catalogo/typespec",
  diagnostics: {
    "fecha-invalida": { severity: "error", messages: { default: paramMessage`La fecha "${"valor"}" no es válida; use el formato AAAA-MM-DD.` } },
    "rango-invalido": { severity: "error", messages: { default: "En @vigencia, 'desde' no puede ser posterior a 'hasta'." } },
    "tipo-invalido": { severity: "error", messages: { default: paramMessage`El tipo "${"valor"}" no es válido.` } },
    "responsable-vacio": { severity: "error", messages: { default: "El responsable no puede estar vacío." } },
  },
  state: {
    responsable: { description: "Responsable del esquema" },
    tipo: { description: "Tipo de esquema" },
    vigencia: { description: "Vigencia del esquema" },
  },
  emitter: {
    options: {
      type: "object",
      additionalProperties: false,
      properties: {
        "main-file": { type: "string", nullable: true, description: "Ruta absoluta del archivo .tsp cuyos tipos se catalogan." },
      },
      required: [],
    },
  },
});

export const { reportDiagnostic, createDiagnostic, stateKeys: keys } = $lib;
