import "@catalogo/typespec";
import "@typespec/http";
import "../../basicos/tipos.tsp";
import "../../comunes/personas.tsp";
import "../../documentos/certificados.tsp";

using Catalogo;
using TypeSpec.Http;
using Chile.Basicos;
using Chile.Comunes;
using Chile.Documentos;

/** API de consulta de personas y emisión de certificados. */
@service(#{ title: "API de Personas" })
@responsable("División de Datos e Interoperabilidad", "interoperabilidad@gobierno.example")
@vigencia("2025-01-01")
namespace Chile.Servicios.Personas {
  /** Error estándar de las APIs del Estado. */
  @error
  model ErrorApi {
    codigo: string;
    mensaje: string;
  }

  @route("/personas")
  interface Personas {
    /** Lista personas, paginado. */
    @get listar(@query pagina?: int32): Pagina<Persona>;

    /** Obtiene una persona por su RUT. */
    @get leer(@path rut: Rut): Persona | ErrorApi;

    /** Registra una persona. */
    @post crear(@body persona: Persona): Persona | ErrorApi;
  }

  @route("/certificados")
  interface Certificados {
    /** Emite un certificado de nacimiento. */
    @route("/nacimiento") @get nacimiento(@query rut: Rut): CertificadoNacimiento | ErrorApi;

    /** Emite un certificado de residencia. */
    @route("/residencia") @get residencia(@query rut: Rut): CertificadoResidencia | ErrorApi;
  }
}
