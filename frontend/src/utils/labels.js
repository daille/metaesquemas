export const RELACION = {
  composicion: 'composición', extension: 'extiende', variante: 'variante', uso: 'parámetro', retorno: 'retorno',
}
export const relLabel = (r) => RELACION[r] || r
export const KIND = { model: 'modelo', scalar: 'escalar', enum: 'enumeración', union: 'unión', interface: 'interfaz' }
export const TIPOS = [
  { id: 'basico', label: 'Dato básico', plural: 'Datos básicos', hint: 'Escalares y enumeraciones' },
  { id: 'estructurado', label: 'Estructurado', plural: 'Estructurados', hint: 'Modelos y uniones' },
  { id: 'documental', label: 'Documental', plural: 'Documentales', hint: 'Documentos oficiales' },
  { id: 'servicio', label: 'Servicio', plural: 'Servicios', hint: 'Contratos de API' },
]
export const fmtDate = (d) => (d ? new Date(d).toLocaleDateString('es-CL', { day: 'numeric', month: 'long', year: 'numeric', timeZone: 'UTC' }) : '')
export const shortName = (id) => id.split('.').pop()
