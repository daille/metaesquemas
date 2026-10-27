<script setup>
import { ref } from 'vue'
import TipoBadge from './TipoBadge.vue'
import { relLabel } from '../utils/labels'

const props = defineProps({ node: { type: Object, required: true }, root: Boolean, level: { type: Number, default: 0 } })
const open = ref(props.level < 2)
</script>

<template>
  <li role="treeitem" :aria-expanded="node.children?.length ? open : undefined">
    <div class="row" :class="{ root }">
      <button v-if="node.children?.length" class="tog" type="button" :aria-label="(open ? 'Contraer ' : 'Expandir ') + node.name" @click="open = !open">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" :style="{ transform: open ? 'rotate(90deg)' : 'none' }" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
      </button>
      <span v-else class="tog" aria-hidden="true" />
      <span v-if="node.relations?.length" class="rel mono">{{ node.relations.map(relLabel).join(' + ') }}</span>
      <RouterLink v-if="node.exists && !root" :to="`/esquemas/${node.id}`" class="nm">{{ node.name }}</RouterLink>
      <strong v-else-if="root" class="nm">{{ node.name }}</strong>
      <span v-else class="nm missing" :title="node.id">{{ node.name }}</span>
      <TipoBadge v-if="node.tipo" :tipo="node.tipo" />
      <span class="ns text-xs mono">{{ node.id }}</span>
      <span v-if="!node.exists" class="badge badge-code">no catalogado</span>
      <span v-if="node.cycle" class="badge badge-amber" title="Este esquema ya aparece más arriba en la rama">referencia circular</span>
      <span v-if="node.truncated" class="badge badge-code" title="Hay más niveles; aumente la profundidad">…</span>
    </div>
    <ul v-if="open && node.children?.length" role="group">
      <DepTreeNode v-for="c in node.children" :key="c.id + c.relation" :node="c" :level="level + 1" />
    </ul>
  </li>
</template>

<style scoped>
ul { list-style: none; margin-left: 13px; padding-left: 12px; border-left: 1px solid var(--border-bright); }
.row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; padding: 4px 0; }
.tog { width: 24px; height: 24px; display: inline-flex; align-items: center; justify-content: center; background: none; border: 0; border-radius: 6px; cursor: pointer; color: var(--text-secondary); flex-shrink: 0; }
.tog:hover { background: var(--bg-hover); }
.tog svg { transition: transform .15s; }
.rel { font-size: 11px; color: var(--text-secondary); background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 4px; padding: 0 6px; }
.nm { font-size: 15px; font-weight: 500; }
.root .nm { font-size: 16px; }
.missing { color: var(--text-secondary); font-style: italic; }
.ns { color: var(--text-muted); }
</style>
