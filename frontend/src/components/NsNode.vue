<script setup>
import { ref } from 'vue'
const props = defineProps({ node: Object, selected: String, level: { type: Number, default: 0 } })
const emit = defineEmits(['select'])
const open = ref(props.level < 1 || (props.selected || '').startsWith(props.node.path))
</script>

<template>
  <li role="treeitem" :aria-expanded="node.children?.length ? open : undefined" :aria-selected="selected === node.path">
    <div class="row" :class="{ sel: selected === node.path }">
      <button v-if="node.children?.length" class="tog" type="button" :aria-label="(open ? 'Contraer ' : 'Expandir ') + node.name" @click="open = !open">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" :style="{ transform: open ? 'rotate(90deg)' : 'none' }" aria-hidden="true"><path d="m9 6 6 6-6 6" /></svg>
      </button>
      <span v-else class="tog" aria-hidden="true" />
      <button class="lbl" type="button" @click="emit('select', selected === node.path ? '' : node.path)">
        <span class="nm">{{ node.name }}</span><span class="cnt mono">{{ node.count }}</span>
      </button>
    </div>
    <ul v-if="open && node.children?.length" role="group">
      <NsNode v-for="c in node.children" :key="c.path" :node="c" :selected="selected" :level="level + 1" @select="emit('select', $event)" />
    </ul>
  </li>
</template>

<style scoped>
ul { list-style: none; margin-left: 10px; padding-left: 8px; border-left: 1px solid var(--border); }
.row { display: flex; align-items: center; border-radius: var(--radius); }
.row.sel { background: var(--accent-dim); }
.tog { width: 22px; height: 22px; display: inline-flex; align-items: center; justify-content: center; background: none; border: 0; border-radius: 6px; cursor: pointer; color: var(--text-secondary); flex-shrink: 0; }
.tog svg { transition: transform .15s; }
.lbl { flex: 1; display: flex; justify-content: space-between; align-items: center; gap: 8px; background: none; border: 0; padding: 4px 8px; cursor: pointer; text-align: left; border-radius: var(--radius); font-size: 14px; }
.lbl:hover { background: var(--bg-hover); }
.cnt { font-size: 11px; color: var(--text-secondary); background: var(--bg-elevated); border: 1px solid var(--border); border-radius: 10px; padding: 0 7px; }
</style>
