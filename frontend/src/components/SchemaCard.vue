<script setup>
import TipoBadge from './TipoBadge.vue'
import VigenciaBadge from './VigenciaBadge.vue'
import Icon from './Icon.vue'
import { KIND } from '../utils/labels'

defineProps({ e: { type: Object, required: true } })
const PROJ = { jsonschema: 'JSON Schema', xsd: 'XSD', proto: 'Protobuf', openapi: 'OpenAPI' }
</script>

<template>
  <article class="card sc">
    <header class="top">
      <TipoBadge :tipo="e.tipo" />
      <VigenciaBadge :estado="e.vigenciaEstado" :vigencia="e.vigencia" />
      <span v-if="e.deprecado" class="badge badge-amber">Obsoleto</span>
      <span v-if="e.plantilla" class="badge badge-code">plantilla</span>
    </header>
    <h2><RouterLink :to="`/esquemas/${e.id}`">{{ e.name }}</RouterLink> <span class="kind mono">{{ KIND[e.kind] || e.kind }}</span></h2>
    <p class="ns mono text-xs">{{ e.namespace }}</p>
    <p v-if="e.doc" class="doc text-muted">{{ e.doc }}</p>
    <dl class="meta text-sm">
      <div><dt><Icon name="user" :size="15" /> Responsable</dt><dd>{{ e.responsable?.nombre || '—' }}</dd></div>
      <div><dt><Icon name="layers" :size="15" /> Usado por</dt><dd><strong>{{ e.usedByCount }}</strong> {{ e.usedByCount === 1 ? 'esquema' : 'esquemas' }}</dd></div>
      <div><dt><Icon name="link" :size="15" /> Depende de</dt><dd><strong>{{ e.dependencyCount }}</strong></dd></div>
    </dl>
    <footer class="proj">
      <span class="badge badge-code">TypeSpec</span>
      <span v-for="p in e.projections.filter((x) => PROJ[x])" :key="p" class="badge badge-code">{{ PROJ[p] }}</span>
    </footer>
  </article>
</template>

<style scoped>
.sc { display: flex; flex-direction: column; gap: 8px; }
.top { display: flex; gap: 6px; flex-wrap: wrap; }
h2 { font-size: 19px; letter-spacing: -.01em; display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
.kind { font-size: 11px; font-weight: 400; color: var(--text-secondary); }
.ns { color: var(--text-secondary); word-break: break-all; }
.doc { display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.meta { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 8px 16px; border-top: 1px solid var(--border); padding-top: 10px; }
.meta dt { display: flex; align-items: center; gap: 5px; color: var(--text-secondary); font-size: 12px; text-transform: uppercase; letter-spacing: .05em; }
.meta dd { margin-top: 2px; }
.proj { display: flex; gap: 6px; flex-wrap: wrap; }
</style>
