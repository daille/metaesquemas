export class ApiError extends Error {
  constructor(status, code, message) { super(message); this.status = status; this.code = code }
}

const BASE = '/api/v1'

function friendly(status, code, msg, retry) {
  if (status === 429) return `Demasiadas solicitudes. Intente nuevamente${retry ? ` en ${retry} s` : ' en unos segundos'}.`
  if (status === 403 && code === 'region_blocked') return 'El acceso no está disponible desde su ubicación.'
  if (status === 503) return 'El servicio está ocupado o no disponible. Reintente en unos segundos.'
  return msg || 'No fue posible completar la solicitud.'
}

async function request(path, params, { signal, text = false } = {}) {
  const q = new URLSearchParams()
  for (const [k, v] of Object.entries(params || {})) if (v !== undefined && v !== null && v !== '') q.set(k, v)
  const url = path.startsWith('/api/') ? path : `${BASE}${path}${q.toString() ? `?${q}` : ''}`
  let res
  try {
    res = await fetch(url, { signal, headers: { Accept: text ? 'text/plain, application/json, application/xml' : 'application/json' } })
  } catch (e) {
    if (e.name === 'AbortError') throw e
    throw new ApiError(0, 'network', 'No hay conexión con el servidor.')
  }
  if (!res.ok) {
    let body = {}
    try { body = await res.json() } catch { /* sin JSON */ }
    throw new ApiError(res.status, body?.error?.code, friendly(res.status, body?.error?.code, body?.error?.message, res.headers.get('Retry-After')))
  }
  return text ? res.text() : res.json()
}

export const api = {
  stats: (o) => request('/stats', {}, o),
  facets: (o) => request('/facets', {}, o),
  namespaces: (o) => request('/namespaces', {}, o),
  list: (p, o) => request('/schemas', p, o),
  schema: (id, version, o) => request(`/schemas/${encodeURIComponent(id)}`, { version }, o),
  tree: (id, direction, depth, o) => request(`/schemas/${encodeURIComponent(id)}/dependencies`, { direction, depth }, o),
  graph: (id, depth, o) => request(`/schemas/${encodeURIComponent(id)}/graph`, { depth }, o),
  /** Texto de una proyección ya generada (la URL viene del detalle del esquema). */
  artifact: (url, o) => request(url, {}, { ...o, text: true }),
}

export const downloadUrl = (url) => `${url}${url.includes('?') ? '&' : '?'}download=1`
