import "@catalogo/typespec";
import "../basicos/tipos.tsp";
import "../comunes/personas.tsp";

using Catalogo;
using Chile.Basicos;
using Chile.Comunes;

/** Documentos oficiales emitidos por el Estado. */
@responsable("Registro Civil e Identificación", "documentos@registro.example")
@vigencia("2024-03-01")
namespace Chile.Documentos;

/** Certificado de nacimiento. */
@tipo(TipoEsquema.documental)
model CertificadoNacimiento {
  folio: string;
  titular: Persona;
  fechaNacimiento: plainDate;
  lugarNacimiento: Direccion;
  fechaEmision: utcDateTime;
}

/** Certificado de residencia. */
@tipo(TipoEsquema.documental)
@responsable("Servicio de Registro Civil – Unidad de Residencia")
model CertificadoResidencia {
  folio: string;
  residente: PersonaResidente;
  fechaEmision: utcDateTime;
  /** Monto pagado por el trámite, si corresponde. */
  arancel?: MontoCLP;
}
