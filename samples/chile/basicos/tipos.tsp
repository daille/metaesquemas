import "@catalogo/typespec";

using Catalogo;

/** Tipos de dato básicos reutilizables por toda la administración. */
@responsable("División de Gobierno Digital", "datos@gobierno.example")
@vigencia("2024-01-01")
namespace Chile.Basicos;

/** Rol Único Nacional, con guion y dígito verificador. */
@pattern("^\\d{7,8}-[\\dkK]$")
@minLength(9)
@maxLength(10)
scalar Rut extends string;

/** Correo electrónico. */
@pattern("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$")
@maxLength(254)
scalar Email extends string;

/** Código numérico de región (1 a 16). */
@minValue(1)
@maxValue(16)
scalar CodigoRegion extends int32;

/** Estado civil de una persona. */
enum EstadoCivil {
  Soltero,
  Casado,
  Viudo,
  Divorciado,
  AcuerdoUnionCivil,
}

/** Monto en pesos chilenos, sin decimales. */
@minValue(0)
scalar MontoCLP extends int64;
