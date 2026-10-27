import "@catalogo/typespec";
import "../basicos/tipos.tsp";

using Catalogo;
using Chile.Basicos;

/** Estructuras comunes de personas y domicilios. */
@responsable("División de Datos e Interoperabilidad", "interoperabilidad@gobierno.example")
@vigencia("2024-06-01")
namespace Chile.Comunes;

/** Domicilio postal o residencial. */
model Direccion {
  /** Nombre de la calle, pasaje o camino. */
  @minLength(1) calle: string;
  numero?: string;
  comuna: string;
  region: CodigoRegion;
}

/** Datos de contacto de una persona. */
model Contacto {
  email?: Email;
  telefonos: string[];
}

/** Persona natural. */
model Persona {
  rut: Rut;
  nombres: string;
  apellidos: string;
  estadoCivil?: EstadoCivil;
  domicilio?: Direccion;
  contacto?: Contacto;
}

/** Persona con un domicilio obligatorio (extiende Persona). */
model PersonaResidente extends Persona {
  domicilio: Direccion;
  fechaResidencia: plainDate;
}

/** Respuesta paginada de cualquier colección. */
model Pagina<T> {
  items: T[];
  total: int32;
  siguiente?: url;
}

/** Contribuyente del sistema tributario anterior (reemplazado por Persona). */
@vigencia("2015-01-01", "2023-12-31")
@responsable("Servicio de Impuestos (histórico)")
model ContribuyenteLegado {
  rut: Rut;
  razonSocial: string;
}
