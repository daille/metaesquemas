import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import fx from './fixtures/api.json'
import { highlightLines } from '../src/utils/highlight'

vi.mock('cytoscape', () => {
  const col = () => ({ length: 0, addClass() {}, removeClass() {}, remove() {} })
  return { default: () => ({ on() {}, add() {}, elements: col, nodes: col, getElementById: col, layout: () => ({ run() {} }), fit() {}, destroy() {} }) }
})

window.scrollTo = () => {}
const P = 'Chile.Comunes.Persona', S = 'Chile.Servicios.Personas.Personas'
let errors, calls

const route = (url) => {
  const u = new URL(url, 'http://x')
  const p = u.pathname.replace('/api/v1', '')
  const q = u.searchParams
  if (p === '/stats') return fx.stats
  if (p === '/facets') return fx.facets
  if (p === '/namespaces') return fx.namespaces
  if (p === '/schemas') return q.get('tipo') === 'basico' ? fx.list_basico : fx.list
  if (p === `/schemas/${P}`) return fx.persona
  if (p === `/schemas/${S}`) return fx.servicio
  if (p === `/schemas/${S}/dependencies`) return fx.tree_uses
  if (p === `/schemas/${P}/dependencies`) return q.get('direction') === 'usedBy' ? fx.tree_persona_usedBy : fx.tree_persona_uses
  if (p === '/schemas/Chile.Basicos.Rut/dependencies') return fx.tree_usedBy
  if (p.endsWith('/graph')) return fx.graph
  let m = p.match(/^\/schemas\/(.+)\/artifacts\/(.+)$/)
  if (m) return m[1] === S && m[2] === 'openapi' ? fx.art_openapi : fx.art[m[2]]
  return undefined
}

beforeEach(() => {
  errors = []; calls = []
  vi.spyOn(console, 'warn').mockImplementation((...a) => errors.push(['warn', ...a]))
  vi.spyOn(console, 'error').mockImplementation((...a) => errors.push(['error', ...a]))
  globalThis.fetch = vi.fn(async (url) => {
    calls.push(String(url))
    const body = route(String(url))
    const ok = body !== undefined
    const text = typeof body === 'string' ? body : JSON.stringify(body ?? { error: { code: 'not_found', message: 'Recurso no encontrado.' } })
    return { ok, status: ok ? 200 : 404, headers: { get: () => null }, json: async () => JSON.parse(text), text: async () => text }
  })
  Object.defineProperty(navigator, 'clipboard', { value: { writeText: vi.fn(async () => {}) }, configurable: true })
})

async function open(path) {
  setActivePinia(createPinia())
  const { createAppRouter } = await import('../src/router')
  const App = (await import('../src/App.vue')).default
  const router = createAppRouter()
  router.push(path); await router.isReady()
  const w = mount(App, { global: { plugins: [createPinia(), router] }, attachTo: document.body })
  await flushPromises(); await flushPromises()
  return { w, router }
}

describe('resaltado', () => {
  it('escapa HTML: un esquema malicioso no puede inyectar marcado', () => {
    const lines = highlightLines('/** <img src=x onerror=alert(1)> */\nmodel A { "<script>alert(1)</script>": string }', 'tsp').join('\n')
    expect(lines).not.toContain('<img')
    expect(lines).not.toContain('<script>')
    expect(lines).toContain('&lt;img')
  })
  it('resalta por línea, incluso en comentarios de varias líneas', () => {
    const l = highlightLines('/* a\nb */\nmodel X {}', 'tsp')
    expect(l).toHaveLength(3)
    expect(l[0]).toContain('tk-c'); expect(l[1]).toContain('tk-c'); expect(l[2]).toContain('tk-k')
    expect(highlightLines('{"a": [1, true, null]}', 'json')[0]).toMatch(/tk-p.*tk-n.*tk-k/)
  })
})

describe('vistas', () => {
  it('inicio: estadísticas por tipo desde la API', async () => {
    const { w } = await open('/')
    expect(w.text()).toContain('Un esquema, todas las tecnologías')
    for (const t of ['Datos básicos', 'Estructurados', 'Documentales', 'Servicios']) expect(w.text()).toContain(t)
    expect(w.text()).toContain(String(fx.stats.total))
    expect(w.text()).toContain(fx.stats.masUsados[0].name)
    expect(errors).toEqual([])
  })

  it('catálogo: lista, filtros en la URL y consulta con los parámetros', async () => {
    const { w, router } = await open('/catalogo')
    expect(w.findAll('article').length).toBe(fx.list.items.length)
    expect(w.text()).toContain('Persona')
    expect(w.text()).toContain('Responsable')
    const chk = w.findAll('input[type=checkbox]').find((c) => c.element.closest('label').textContent.includes('Dato básico'))
    await chk.setValue(true); await flushPromises(); await flushPromises()
    expect(router.currentRoute.value.query.tipo).toBe('basico')
    expect(calls.some((u) => u.includes('tipo=basico'))).toBe(true)
    expect(w.findAll('article').length).toBe(fx.list_basico.items.length)
    await w.find('select#c-vig').setValue('vigente'); await flushPromises()
    expect(router.currentRoute.value.query.vigencia).toBe('vigente')
    await w.find('input[name=usada][type=radio]:not(:checked)').setValue(true); await flushPromises()
    expect(router.currentRoute.value.query.usada).toBeTruthy()
    expect(errors).toEqual([])
  })

  it('catálogo: la búsqueda de texto actualiza la URL (con retardo) y se puede limpiar', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const { w, router } = await open('/catalogo')
    await w.find('#c-q').setValue('persona')
    vi.advanceTimersByTime(400); vi.useRealTimers(); await flushPromises()
    expect(router.currentRoute.value.query.q).toBe('persona')
    expect(calls.some((u) => u.includes('q=persona'))).toBe(true)
    expect(errors).toEqual([])
  })

  it('detalle: metadatos, TypeSpec y pestañas de proyecciones disponibles', async () => {
    const { w } = await open(`/esquemas/${P}`)
    expect(w.find('h1').text()).toBe('Persona')
    expect(w.text()).toContain('División de Datos e Interoperabilidad')
    expect(w.text()).toContain('interoperabilidad@gobierno.example')
    expect(w.text()).toContain('Persona natural.')
    expect(w.text()).toContain('model Persona')           // fragmento TypeSpec ya cargado
    const tab = (label) => w.findAll('[role=tab]').find((t) => t.text().startsWith(label))
    expect(tab('JSON Schema').attributes('disabled')).toBeUndefined()
    expect(tab('OpenAPI').attributes('disabled')).toBeDefined() // un modelo no tiene OpenAPI propio
    await tab('JSON Schema').trigger('click'); await flushPromises(); await flushPromises()
    expect(w.text()).toContain('$defs')
    await tab('XSD').trigger('click'); await flushPromises(); await flushPromises()
    expect(w.text()).toContain('xs:complexType')
    await tab('Protobuf').trigger('click'); await flushPromises(); await flushPromises()
    expect(w.text()).toContain('syntax')
    expect(errors).toEqual([])
  })

  it('detalle: archivo completo, estructura, dependencias y usado por', async () => {
    const { w } = await open(`/esquemas/${P}`)
    await w.findAll('.filter-pill').find((b) => b.text().includes('Archivo completo')).trigger('click'); await flushPromises(); await flushPromises()
    expect(w.text()).toContain('namespace Chile.Comunes')
    const go = async (label) => { await w.findAll('[role=tab]').find((t) => t.text().startsWith(label)).trigger('click'); await flushPromises(); await flushPromises() }
    await go('Estructura'); expect(w.text()).toContain('estadoCivil'); expect(w.text()).toContain('Rut')
    await go('Usado por'); expect(w.text()).toMatch(/esquemas? lo usan?/); expect(w.find('.alert-error').exists()).toBe(false)
    await go('Dependencias'); expect(w.find('[role=tree]').exists()).toBe(true); expect(w.find('.alert-error').exists()).toBe(false); expect(w.text()).toContain('Direccion')
    await go('Grafo'); expect(w.text()).toContain('Ver como tabla')
    await go('Versiones'); expect(w.text()).toContain('Versión 1')
    expect(errors).toEqual([])
  })

  it('detalle de un servicio: OpenAPI disponible, JSON Schema no', async () => {
    const { w, router } = await open(`/esquemas/${S}?tab=openapi`)
    expect(w.text()).toContain('API de Personas')
    const tab = (label) => w.findAll('[role=tab]').find((t) => t.text().startsWith(label))
    expect(tab('OpenAPI').attributes('disabled')).toBeUndefined()
    expect(tab('JSON Schema').attributes('disabled')).toBeDefined()
    expect(router.currentRoute.value.query.tab).toBe('openapi')
    expect(w.text()).toContain('"paths"')
    expect(errors).toEqual([])
  })

  it('detalle: esquema inexistente muestra un mensaje, no una página rota', async () => {
    const { w } = await open('/esquemas/No.Existe')
    expect(w.text()).toContain('no existe')
    expect(errors).toEqual([])
  })

  it('API: documentación con ejemplos', async () => {
    const { w } = await open('/api')
    expect(w.text()).toContain('Endpoints'); expect(w.text()).toContain('/schemas/{id}/artifacts/{kind}')
    await w.findAll('[role=tab]').find((t) => t.text() === 'Go').trigger('click')
    expect(w.text()).toContain('http.Client')
    expect(w.text()).toContain('Ingesta de esquemas')
    expect(errors).toEqual([])
  })
})
