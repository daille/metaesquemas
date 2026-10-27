<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api/client'
import SchemaCard from '../components/SchemaCard.vue'
import NsNode from '../components/NsNode.vue'
import { TIPOS } from '../utils/labels'

const route = useRoute()
const router = useRouter()
const PAGE = 12

const facets = ref(null)
const namespaces = ref([])
const result = ref({ items: [], total: 0 })
const loading = ref(false)
const error = ref('')
const text = ref(route.query.q || '')
let ctrl, timer

// Todo el estado de los filtros vive en la URL: se puede compartir y funciona con "atrás".
const f = computed(() => ({
  q: route.query.q || '', tipos: route.query.tipo ? String(route.query.tipo).split(',') : [], responsable: route.query.responsable || '',
  vigencia: route.query.vigencia || '', usada: route.query.usada || '', namespace: route.query.namespace || '', sort: route.query.sort || 'nombre',
  page: Math.max(1, Number(route.query.page) || 1),
}))
function setQ(patch) {
  const q = { ...route.query, ...patch }
  if (!('page' in patch)) delete q.page
  for (const k of Object.keys(q)) if (q[k] === '' || q[k] === undefined || (Array.isArray(q[k]) && !q[k].length)) delete q[k]
  router.replace({ query: q })
}
const toggleTipo = (id) => {
  const s = new Set(f.value.tipos)
  s.has(id) ? s.delete(id) : s.add(id)
  setQ({ tipo: [...s].join(',') })
}
watch(text, (v) => { clearTimeout(timer); timer = setTimeout(() => v !== f.value.q && setQ({ q: v.trim() }), 300) })
watch(() => route.query.q, (v) => { if ((v || '') !== text.value.trim()) text.value = v || '' })
onBeforeUnmount(() => clearTimeout(timer))

async function load() {
  ctrl?.abort(); ctrl = new AbortController()
  loading.value = true; error.value = ''
  const x = f.value
  try {
    result.value = await api.list({ q: x.q, tipo: x.tipos.join(','), responsable: x.responsable, vigencia: x.vigencia, usada: x.usada, namespace: x.namespace, sort: x.sort === 'nombre' ? '' : x.sort, limit: PAGE, offset: (x.page - 1) * PAGE }, { signal: ctrl.signal })
  } catch (e) { if (e.name !== 'AbortError') { error.value = e.message; result.value = { items: [], total: 0 } } } finally { loading.value = false }
}
watch(() => route.query, load, { immediate: true })
onMounted(async () => {
  try { [facets.value, namespaces.value] = await Promise.all([api.facets(), api.namespaces().then((r) => r.items)]) } catch (e) { error.value = e.message }
})

const pages = computed(() => Math.max(1, Math.ceil(result.value.total / PAGE)))
const countOf = (list, v) => list?.find((x) => x.value === v)?.count ?? 0
const active = computed(() => [f.value.q, f.value.tipos.length, f.value.responsable, f.value.vigencia, f.value.usada, f.value.namespace].filter(Boolean).length)
const clear = () => { text.value = ''; router.replace({ query: {} }) }
const vigLabels = { vigente: 'Vigente', vencida: 'Vencida', futura: 'Vigencia futura', 'sin-vigencia': 'Sin vigencia definida' }
</script>

<template>
  <div class="container page">
    <h1 class="page-title">Catálogo de esquemas</h1>
    <p class="page-subtitle">Busque entre los esquemas del repositorio. Cada resultado indica su responsable, su vigencia y cuántos otros esquemas lo utilizan.</p>

    <div class="topbar">
      <div class="form-group" style="margin: 0; flex: 1; min-width: 260px">
        <label class="form-label" for="c-q">Buscar</label>
        <input id="c-q" v-model="text" class="input" type="search" placeholder="Nombre, descripción, namespace o responsable" autocomplete="off" />
      </div>
      <div class="form-group" style="margin: 0">
        <label class="form-label" for="c-sort">Ordenar por</label>
        <select id="c-sort" class="input" :value="f.sort" @change="setQ({ sort: $event.target.value === 'nombre' ? '' : $event.target.value })">
          <option value="nombre">Nombre</option><option value="usos">Más utilizados</option><option value="responsable">Responsable</option><option value="namespace">Namespace</option>
        </select>
      </div>
    </div>

    <div class="layout">
      <aside class="filters" aria-label="Filtros">
        <details open>
          <summary class="fs">Filtros <span v-if="active" class="badge badge-accent">{{ active }}</span></summary>

          <fieldset class="grp">
            <legend class="form-label">Tipo de esquema</legend>
            <label v-for="t in TIPOS" :key="t.id" class="chk">
              <input type="checkbox" :checked="f.tipos.includes(t.id)" @change="toggleTipo(t.id)" />
              <span>{{ t.label }}</span><span class="cnt mono">{{ countOf(facets?.tipos, t.id) }}</span>
            </label>
          </fieldset>

          <div class="grp">
            <label class="form-label" for="c-resp">Responsable</label>
            <select id="c-resp" class="input" :value="f.responsable" @change="setQ({ responsable: $event.target.value })">
              <option value="">Todos</option>
              <option v-for="r in facets?.responsables || []" :key="r.value" :value="r.value">{{ r.value }} ({{ r.count }})</option>
            </select>
          </div>

          <div class="grp">
            <label class="form-label" for="c-vig">Vigencia</label>
            <select id="c-vig" class="input" :value="f.vigencia" @change="setQ({ vigencia: $event.target.value })">
              <option value="">Cualquiera</option>
              <option v-for="v in facets?.vigencia || []" :key="v.value" :value="v.value">{{ vigLabels[v.value] || v.value }} ({{ v.count }})</option>
            </select>
          </div>

          <fieldset class="grp">
            <legend class="form-label">Usado por otros esquemas</legend>
            <label class="chk"><input type="radio" name="usada" :checked="!f.usada" @change="setQ({ usada: '' })" /><span>Todos</span></label>
            <label class="chk"><input type="radio" name="usada" :checked="f.usada === 'si'" @change="setQ({ usada: 'si' })" /><span>Sí, es reutilizado</span><span class="cnt mono">{{ countOf(facets?.usada, 'si') }}</span></label>
            <label class="chk"><input type="radio" name="usada" :checked="f.usada === 'no'" @change="setQ({ usada: 'no' })" /><span>No lo usa ninguno</span><span class="cnt mono">{{ countOf(facets?.usada, 'no') }}</span></label>
          </fieldset>

          <div class="grp">
            <span class="form-label" id="ns-label">Namespace</span>
            <ul v-if="namespaces.length" role="tree" aria-labelledby="ns-label" class="ns">
              <NsNode v-for="n in namespaces" :key="n.path" :node="n" :selected="f.namespace" @select="setQ({ namespace: $event })" />
            </ul>
            <p v-else class="text-sm text-muted">Sin namespaces.</p>
          </div>

          <button class="btn btn-sm" type="button" :disabled="!active" @click="clear">Limpiar filtros</button>
        </details>
      </aside>

      <section aria-live="polite" :aria-busy="loading">
        <p class="count text-sm text-muted" role="status">{{ loading ? 'Buscando…' : `${result.total} ${result.total === 1 ? 'esquema' : 'esquemas'}` }}<template v-if="f.namespace"> en <span class="mono">{{ f.namespace }}</span></template></p>
        <div v-if="error" class="alert alert-error" role="alert">{{ error }}</div>
        <div v-else-if="loading && !result.items.length" class="grid"><div v-for="i in 4" :key="i" class="skeleton" style="height: 210px" /></div>
        <p v-else-if="!result.items.length" class="empty">No hay esquemas que coincidan con los filtros. <button class="btn btn-sm" type="button" style="margin-left: 8px" @click="clear">Limpiar filtros</button></p>
        <div v-else class="grid"><SchemaCard v-for="e in result.items" :key="e.id" :e="e" /></div>

        <nav v-if="pages > 1" class="pager" aria-label="Paginación">
          <button class="btn btn-sm" type="button" :disabled="f.page <= 1" @click="setQ({ page: f.page - 1 })">← Anterior</button>
          <span class="text-sm">Página {{ f.page }} de {{ pages }}</span>
          <button class="btn btn-sm" type="button" :disabled="f.page >= pages" @click="setQ({ page: f.page + 1 })">Siguiente →</button>
        </nav>
      </section>
    </div>
  </div>
</template>

<style scoped>
.topbar { display: flex; gap: 16px; align-items: flex-end; flex-wrap: wrap; margin: 24px 0; }
.layout { display: grid; grid-template-columns: 280px minmax(0, 1fr); gap: 28px; align-items: start; }
.filters { position: sticky; top: calc(var(--header-h) + 16px); background: var(--bg-surface); border: 1px solid var(--border); border-radius: var(--radius-lg); padding: 16px; max-height: calc(100vh - var(--header-h) - 32px); overflow: auto; }
.fs { font-weight: 600; cursor: pointer; margin-bottom: 12px; display: flex; align-items: center; gap: 8px; }
.grp { border: 0; margin-bottom: 18px; display: flex; flex-direction: column; gap: 6px; }
.chk { display: flex; align-items: center; gap: 8px; cursor: pointer; font-size: 15px; }
.chk span:nth-of-type(1) { flex: 1; }
.chk input { width: 17px; height: 17px; accent-color: var(--accent); }
.cnt { font-size: 11px; color: var(--text-secondary); background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 10px; padding: 0 7px; }
.ns { list-style: none; }
.count { margin-bottom: 12px; }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(330px, 1fr)); gap: 18px; }
.pager { display: flex; justify-content: center; align-items: center; gap: 16px; margin-top: 28px; }
@media (max-width: 900px) { .layout { grid-template-columns: 1fr; } .filters { position: static; max-height: none; } }
</style>
