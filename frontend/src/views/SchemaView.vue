<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api, downloadUrl } from '../api/client'
import CodeViewer from '../components/CodeViewer.vue'
import TipoBadge from '../components/TipoBadge.vue'
import VigenciaBadge from '../components/VigenciaBadge.vue'
import MembersTable from '../components/MembersTable.vue'
import DepTree from '../components/DepTree.vue'
import DepGraph from '../components/DepGraph.vue'
import { LANG_OF } from '../utils/highlight'
import { KIND, fmtDate, relLabel } from '../utils/labels'

const props = defineProps({ id: { type: String, required: true } })
const route = useRoute()
const router = useRouter()

const d = ref(null)
const loading = ref(false)
const error = ref('')
const notFound = ref(false)
const texts = ref({})       // contenido de cada proyección, cacheado por pestaña y versión
const tspScope = ref('tsp') // "tsp" (solo este esquema) | "tsp-file" (archivo completo)

const version = computed(() => Number(route.query.version) || 0)
const tab = computed(() => route.query.tab || 'tsp')
const setQuery = (patch) => router.replace({ query: { ...route.query, ...patch } })

async function load() {
  loading.value = true; error.value = ''; notFound.value = false; d.value = null; texts.value = {}
  try { d.value = await api.schema(props.id, version.value || undefined) } catch (e) { e.status === 404 ? (notFound.value = true) : (error.value = e.message) } finally { loading.value = false }
}
watch(() => [props.id, version.value], load, { immediate: true })

const PROJ = [['tsp', 'TypeSpec'], ['jsonschema', 'JSON Schema'], ['xsd', 'XSD'], ['proto', 'Protobuf'], ['openapi', 'OpenAPI']]
const tabs = computed(() => {
  const av = d.value?.available || {}
  return [
    ...PROJ.map(([id, label]) => ({ id, label, disabled: !av[id], kind: 'proj' })),
    { id: 'estructura', label: 'Estructura' }, { id: 'dependencias', label: 'Dependencias' },
    { id: 'usadopor', label: `Usado por (${d.value?.usedByCount ?? 0})` }, { id: 'grafo', label: 'Grafo' }, { id: 'versiones', label: 'Versiones' },
  ]
})
const known = computed(() => new Set((d.value?.dependencies || []).filter((x) => x.exists).map((x) => x.id)))

// Carga perezosa del texto de la proyección activa (lectura directa del repositorio, sin compilar).
const artKey = computed(() => (tab.value === 'tsp' ? tspScope.value : tab.value))
const cur = computed(() => texts.value[artKey.value] || { text: '', loading: false, error: '' })
async function ensureText() {
  const k = artKey.value, url = d.value?.available?.[k]
  if (!url || texts.value[k]?.text || texts.value[k]?.loading) return
  texts.value = { ...texts.value, [k]: { text: '', loading: true, error: '' } }
  try { texts.value = { ...texts.value, [k]: { text: await api.artifact(url), loading: false, error: '' } } } catch (e) { texts.value = { ...texts.value, [k]: { text: '', loading: false, error: e.message } } }
}
watch(() => [d.value, artKey.value], () => { if (d.value && PROJ.some(([id]) => id === tab.value)) ensureText() }, { immediate: true })

const fileName = computed(() => {
  const ext = { tsp: 'tsp', 'tsp-file': 'tsp', jsonschema: 'schema.json', xsd: 'xsd', proto: 'proto', openapi: 'openapi.json' }[artKey.value]
  return artKey.value === 'tsp-file' ? d.value?.file?.split('/').pop() : `${props.id}.${ext}`
})
const projNote = computed(() => {
  if (!d.value) return ''
  if (d.value.plantilla) return 'Es una plantilla genérica: se proyecta cuando se instancia con tipos concretos.'
  if (d.value.kind === 'interface') return 'Los servicios se proyectan a OpenAPI y a un servicio Protobuf.'
  if (d.value.kind === 'scalar') return 'Los escalares son alias de tipos: no generan mensaje Protobuf ni documento OpenAPI propios.'
  return 'Esta proyección no está disponible para este tipo de esquema.'
})
const contactHref = computed(() => (d.value?.responsable?.contacto?.includes('@') ? `mailto:${d.value.responsable.contacto}` : ''))
const isOld = computed(() => d.value && !d.value.isCurrent)
</script>

<template>
  <div class="container page">
    <nav class="text-sm crumbs" aria-label="Ruta de navegación"><RouterLink to="/catalogo">Catálogo</RouterLink><span aria-hidden="true">/</span><span class="mono">{{ id }}</span></nav>

    <div v-if="loading" aria-busy="true"><div class="skeleton" style="height: 44px; width: 40%; margin: 16px 0" /><div class="skeleton" style="height: 240px" /></div>
    <div v-else-if="error" class="alert alert-error" role="alert" style="margin-top: 16px">{{ error }}</div>
    <div v-else-if="notFound" class="empty" style="margin-top: 16px">
      <p><strong>El esquema no existe{{ version ? ' en la versión indicada' : '' }}.</strong></p>
      <p style="margin-top: 12px"><button v-if="version" class="btn btn-primary" type="button" @click="setQuery({ version: undefined })">Ver la versión vigente</button> <RouterLink class="btn" to="/catalogo">Volver al catálogo</RouterLink></p>
    </div>

    <template v-else-if="d">
      <header class="head">
        <div class="badges"><TipoBadge :tipo="d.tipo" /><VigenciaBadge :estado="d.vigenciaEstado" :vigencia="d.vigencia" detail /><span v-if="d.deprecado" class="badge badge-amber">Obsoleto</span><span class="badge badge-code">{{ KIND[d.kind] || d.kind }}</span></div>
        <h1 class="page-title">{{ d.name }}</h1>
        <p class="mono ns text-sm">{{ d.id }}</p>
        <p v-if="d.doc" class="page-subtitle">{{ d.doc }}</p>
      </header>

      <div v-if="isOld" class="alert alert-warn" role="status" style="margin-bottom: 16px">
        <span>Está viendo la <strong>versión {{ d.requestedVersion }}</strong>, que ya no es la vigente. Las dependencias se muestran según esa versión; «usado por» corresponde a hoy.</span>
        <button class="btn btn-sm" type="button" @click="setQuery({ version: undefined })">Ir a la vigente</button>
      </div>

      <section class="meta card" aria-label="Metadatos del esquema">
        <div><span class="k">Responsable</span><span class="v"><strong>{{ d.responsable?.nombre }}</strong><template v-if="d.responsable?.heredado"> <span class="text-xs text-muted">(heredado de {{ d.responsable.origen }})</span></template>
          <a v-if="contactHref" :href="contactHref" class="block text-sm">{{ d.responsable.contacto }}</a><span v-else-if="d.responsable?.contacto" class="block text-sm">{{ d.responsable.contacto }}</span></span></div>
        <div><span class="k">Vigencia</span><span class="v">{{ d.vigencia ? `${d.vigencia.desde} → ${d.vigencia.hasta || 'sin término'}` : 'No definida' }}<span v-if="d.vigencia?.heredada" class="text-xs text-muted block">heredada del namespace</span></span></div>
        <div><span class="k">Namespace</span><span class="v mono text-sm">{{ d.namespace }}</span></div>
        <div><span class="k">Archivo fuente</span><span class="v mono text-sm">{{ d.file }}</span></div>
        <div><span class="k">Versión</span>
          <span class="v"><select class="input sel" aria-label="Versión del archivo" :value="d.requestedVersion" @change="setQuery({ version: Number($event.target.value) === d.fileInfo?.version ? undefined : $event.target.value })">
            <option v-for="v in [...d.versions].reverse()" :key="v.version" :value="v.version">v{{ v.version }}{{ v.version === d.fileInfo?.version ? ' (vigente)' : '' }} · {{ fmtDate(v.createdAt) }}</option></select></span></div>
        <div><span class="k">Reutilización</span><span class="v"><strong>{{ d.usedByCount }}</strong> {{ d.usedByCount === 1 ? 'esquema lo usa' : 'esquemas lo usan' }} · depende de <strong>{{ d.dependencyCount }}</strong></span></div>
      </section>

      <div class="tabs" role="tablist" aria-label="Proyecciones y análisis" style="margin-top: 24px">
        <button v-for="t in tabs" :id="'t-' + t.id" :key="t.id" class="tab" role="tab" :aria-selected="tab === t.id" :aria-disabled="t.disabled || undefined" :disabled="t.disabled" :title="t.disabled ? projNote : undefined" @click="setQuery({ tab: t.id })">{{ t.label }}</button>
      </div>

      <section role="tabpanel" :aria-labelledby="'t-' + tab">
        <template v-if="PROJ.some(([id]) => id === tab)">
          <div v-if="tab === 'tsp'" class="scope" role="group" aria-label="Alcance del código TypeSpec">
            <button class="filter-pill" type="button" :aria-pressed="tspScope === 'tsp'" @click="tspScope = 'tsp'">Solo este esquema</button>
            <button class="filter-pill" type="button" :aria-pressed="tspScope === 'tsp-file'" @click="tspScope = 'tsp-file'">Archivo completo</button>
          </div>
          <p v-if="tab !== 'tsp'" class="text-sm text-muted" style="margin-bottom: 10px">Generado automáticamente al cargar el archivo fuente: este contenido se lee tal cual, sin compilar.</p>
          <CodeViewer :text="cur.text" :loading="cur.loading" :error="cur.error" :lang="LANG_OF[artKey]" :file-name="fileName" :label="fileName" :download-href="d.available[artKey] ? downloadUrl(d.available[artKey]) : ''" />
        </template>

        <MembersTable v-else-if="tab === 'estructura'" :d="d" />

        <template v-else-if="tab === 'dependencias'">
          <p class="text-muted" style="margin-bottom: 14px">Esquemas de los que depende <strong>{{ d.name }}</strong>, por composición, extensión o uso.</p>
          <DepTree :schema-id="id" direction="uses" />
        </template>

        <template v-else-if="tab === 'usadopor'">
          <p class="text-muted" style="margin-bottom: 14px">Esquemas que utilizan a <strong>{{ d.name }}</strong>: <strong>{{ d.usedByCount }}</strong> en total.</p>
          <ul v-if="d.usedBy?.length" class="used card">
            <li v-for="u in [...new Set(d.usedBy.map((x) => x.id))]" :key="u"><RouterLink :to="`/esquemas/${u}`" class="mono">{{ u }}</RouterLink><span class="rels">{{ d.usedBy.filter((x) => x.id === u).map((x) => relLabel(x.relation)).join(' + ') }}</span></li>
          </ul>
          <DepTree :schema-id="id" direction="usedBy" />
        </template>

        <DepGraph v-else-if="tab === 'grafo'" :schema-id="id" />

        <div v-else-if="tab === 'versiones'">
          <p class="text-muted" style="margin-bottom: 14px">Historial del archivo <span class="mono">{{ d.file }}</span>. Cada carga genera una versión nueva e inmutable.</p>
          <ol class="vers">
            <li v-for="v in [...d.versions].reverse()" :key="v.version" class="card">
              <strong>Versión {{ v.version }}</strong> <span v-if="v.version === d.fileInfo?.version" class="badge badge-green">Vigente</span>
              <p class="text-sm text-muted">{{ fmtDate(v.createdAt) }} · {{ v.motivo === 'carga' ? 'Carga del archivo' : v.motivo }}</p>
              <button class="btn btn-sm" type="button" style="margin-top: 8px" :disabled="v.version === d.requestedVersion" @click="setQuery({ version: v.version === d.fileInfo?.version ? undefined : v.version, tab: 'tsp' })">{{ v.version === d.requestedVersion ? 'Viendo esta versión' : 'Ver esta versión' }}</button>
            </li>
          </ol>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.crumbs { display: flex; gap: 8px; color: var(--text-secondary); flex-wrap: wrap; }
.head { margin: 16px 0 20px; }
.badges { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 10px; }
.ns { color: var(--text-secondary); word-break: break-all; margin-top: 4px; }
.meta { display: grid; grid-template-columns: repeat(auto-fit, minmax(230px, 1fr)); gap: 16px 28px; }
.meta .k { display: block; font-size: 12px; text-transform: uppercase; letter-spacing: .06em; color: var(--text-secondary); margin-bottom: 3px; font-weight: 600; }
.meta .v { font-size: 15px; word-break: break-word; }
.block { display: block; }
.sel { padding: 5px 10px; font-size: 14px; }
.scope { display: flex; gap: 8px; margin-bottom: 12px; }
.used { list-style: none; margin-bottom: 18px; padding: 6px 18px; }
.used li { display: flex; justify-content: space-between; gap: 12px; padding: 8px 0; border-bottom: 1px solid var(--border); flex-wrap: wrap; }
.used li:last-child { border-bottom: 0; }
.rels { color: var(--text-secondary); font-size: 13px; }
.vers { list-style: none; display: flex; flex-direction: column; gap: 12px; }
.tab:disabled { opacity: .5; cursor: not-allowed; }
</style>
