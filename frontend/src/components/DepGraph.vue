<script setup>
import { ref, watch, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRouter } from 'vue-router'
import cytoscape from 'cytoscape'
import { api } from '../api/client'
import { relLabel } from '../utils/labels'

const props = defineProps({ schemaId: { type: String, required: true } })
const router = useRouter()
const el = ref(null)
const depth = ref(1)
const loading = ref(false)
const error = ref('')
const asTable = ref(false)
const data = ref({ nodes: [], edges: [] })
let cy

const TIPO_COLOR = { basico: '#0a7352', estructurado: '#0b6f8c', documental: '#6639b0', servicio: '#8f5300' }
const REL_STYLE = {
  extension: { color: '#0f1b2d', style: 'solid', width: 3 }, composicion: { color: '#0b6f8c', style: 'solid', width: 2 },
  variante: { color: '#6639b0', style: 'dashed', width: 2 }, retorno: { color: '#8f5300', style: 'dashed', width: 2 }, uso: { color: '#5a6a81', style: 'dotted', width: 2 },
}
const legendTipos = Object.entries({ basico: 'Dato básico', estructurado: 'Estructurado', documental: 'Documental', servicio: 'Servicio' })
const nameOf = (id) => data.value.nodes.find((n) => n.id === id)?.name || id
const rows = computed(() => data.value.edges.map((e) => ({ from: nameOf(e.source), rel: (e.relations || [e.relation]).map(relLabel).join(' + '), to: nameOf(e.target) })))

function styleSheet() {
  const s = [
    { selector: 'node', style: { label: 'data(label)', 'font-family': 'Outfit, sans-serif', 'font-size': 12, color: '#0f1b2d', 'text-valign': 'bottom', 'text-margin-y': 6, 'text-wrap': 'ellipsis', 'text-max-width': 130, 'background-color': '#fff', 'border-width': 3, 'border-color': 'data(color)', width: 28, height: 28 } },
    { selector: 'node[?missing]', style: { shape: 'round-rectangle', 'border-style': 'dashed', 'border-color': '#5a6a81' } },
    { selector: 'node.center', style: { width: 44, height: 44, 'background-color': 'data(color)', 'font-weight': 700, 'font-size': 13 } },
    { selector: 'edge', style: { width: 2, 'curve-style': 'bezier', 'line-color': '#5a6a81', 'target-arrow-color': '#5a6a81', 'target-arrow-shape': 'triangle', 'arrow-scale': 1.1 } },
  ]
  for (const [k, v] of Object.entries(REL_STYLE)) s.push({ selector: `edge[relation = "${k}"]`, style: { 'line-color': v.color, 'target-arrow-color': v.color, 'line-style': v.style, width: v.width } })
  return s
}

async function load() {
  if (!cy) return
  loading.value = true; error.value = ''
  try {
    const g = await api.graph(props.schemaId, depth.value)
    data.value = g
    cy.elements().remove()
    cy.add([
      ...g.nodes.map((n) => ({ group: 'nodes', data: { id: n.id, label: n.name, color: TIPO_COLOR[n.tipo] || '#5a6a81', missing: !n.exists }, classes: n.id === g.center ? 'center' : '' })),
      ...g.edges.map((e) => ({ group: 'edges', data: { id: `${e.source}|${e.target}`, source: e.source, target: e.target, relation: e.relation } })),
    ])
    cy.layout({ name: 'cose', animate: false, fit: true, padding: 40, nodeRepulsion: () => 14000, idealEdgeLength: () => 120 }).run()
  } catch (e) { error.value = e.message } finally { loading.value = false }
}

onMounted(() => {
  cy = cytoscape({ container: el.value, elements: [], style: styleSheet(), wheelSensitivity: 0.25, minZoom: 0.3, maxZoom: 2.5 })
  cy.on('tap', 'node', (ev) => { const id = ev.target.id(); if (id !== props.schemaId && !ev.target.data('missing')) router.push(`/esquemas/${id}`) })
  load()
})
onBeforeUnmount(() => cy?.destroy())
watch(() => [props.schemaId, depth.value], load)
</script>

<template>
  <div>
    <div class="bar">
      <div class="ctl">
        <label class="form-label" for="gdepth">Alcance</label>
        <select id="gdepth" v-model.number="depth" class="input" style="width: auto">
          <option :value="1">Vecinos directos</option><option :value="2">2 saltos</option><option :value="3">3 saltos</option>
        </select>
      </div>
      <div class="ctl">
        <button class="btn btn-sm" type="button" :aria-pressed="asTable" @click="asTable = !asTable">{{ asTable ? 'Ver grafo' : 'Ver como tabla' }}</button>
        <button class="btn btn-sm" type="button" :disabled="asTable" @click="cy && cy.fit(undefined, 40)">Ajustar vista</button>
      </div>
    </div>
    <p class="text-sm text-muted" style="margin-bottom: 10px">Las flechas van del esquema que usa hacia el esquema usado. Haga clic en un nodo para abrirlo.</p>
    <div v-if="error" class="alert alert-error" role="alert">{{ error }}</div>
    <div v-show="!asTable" class="canvas-wrap">
      <div ref="el" class="canvas" role="img" aria-label="Grafo de composición del esquema. Use la vista de tabla como alternativa accesible." />
      <div v-if="loading" class="loading" role="status">Cargando relaciones…</div>
    </div>
    <div v-if="asTable" class="table-wrap">
      <table>
        <caption class="sr-only">Relaciones del esquema</caption>
        <thead><tr><th>Esquema que usa</th><th>Relación</th><th>Esquema usado</th></tr></thead>
        <tbody><tr v-for="(r, i) in rows" :key="i"><td>{{ r.from }}</td><td>{{ r.rel }}</td><td>{{ r.to }}</td></tr></tbody>
      </table>
    </div>
    <ul class="legend" aria-label="Leyenda de tipos">
      <li v-for="[k, l] in legendTipos" :key="k"><span class="dot" :style="{ borderColor: TIPO_COLOR[k] }" />{{ l }}</li>
      <li><span class="dot sq" />No catalogado</li>
    </ul>
    <ul class="legend" aria-label="Leyenda de relaciones">
      <li v-for="(v, k) in REL_STYLE" :key="k"><svg width="34" height="10" aria-hidden="true"><line x1="1" y1="5" x2="33" y2="5" :stroke="v.color" :stroke-width="v.width" :stroke-dasharray="v.style === 'dashed' ? '6 4' : v.style === 'dotted' ? '2 4' : ''" /></svg>{{ relLabel(k) }}</li>
    </ul>
  </div>
</template>

<style scoped>
.bar { display: flex; justify-content: space-between; align-items: flex-end; gap: 12px; flex-wrap: wrap; margin-bottom: 10px; }
.ctl { display: flex; align-items: center; gap: 8px; }
.canvas-wrap { position: relative; border: 1px solid var(--border-bright); border-radius: var(--radius-lg); overflow: hidden; background-color: #fff; background-image: radial-gradient(circle, rgba(185, 197, 214, .8) 1px, transparent 1px); background-size: 28px 28px; }
.canvas { height: 480px; width: 100%; }
.loading { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: rgba(255, 255, 255, .7); color: var(--text-secondary); }
.legend { list-style: none; display: flex; flex-wrap: wrap; gap: 8px 20px; margin-top: 12px; font-size: 13px; color: var(--text-secondary); }
.legend li { display: inline-flex; align-items: center; gap: 8px; }
.dot { width: 14px; height: 14px; border-radius: 50%; border: 3px solid; background: #fff; display: inline-block; }
.dot.sq { border-radius: 3px; border-style: dashed; border-color: #5a6a81; }
</style>
