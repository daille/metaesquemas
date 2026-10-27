import "../src/decorators.js";

using TypeSpec.Reflection;

namespace Catalogo;

/** Clasificación de un esquema en el catálogo. Si no se indica, se infiere (scalar/enum → basico, model → estructurado, interface → servicio). */
enum TipoEsquema {
  basico,
  estructurado,
  documental,
  servicio,
}

/**
 * Responsable de administrar el esquema (unidad o persona). Se hereda desde el namespace
 * si el tipo no lo declara.
 * @param nombre Unidad o persona responsable.
 * @param contacto Correo u otro medio de contacto (opcional).
 */
extern dec responsable(
  target: Model | Scalar | Enum | Union | Interface | Namespace,
  nombre: valueof string,
  contacto?: valueof string
);

/** Tipo de esquema para el catálogo. */
extern dec tipo(target: Model | Scalar | Enum | Union | Interface, valor: TipoEsquema);

/**
 * Período de vigencia (fechas AAAA-MM-DD). Sin `hasta`, la vigencia no tiene término.
 */
extern dec vigencia(
  target: Model | Scalar | Enum | Union | Interface | Namespace,
  desde: valueof string,
  hasta?: valueof string
);
