<script setup>
import { computed, ref } from 'vue'
import { highlightLines } from '../utils/highlight'

const props = defineProps({
  text: { type: String, default: '' }, lang: { type: String, default: 'tsp' }, loading: Boolean, error: { type: String, default: '' },
  fileName: { type: String, default: '' }, downloadHref: { type: String, default: '' }, label: { type: String, default: '' },
})
const copied = ref(false)
const lines = computed(() => (props.text ? highlightLines(props.text, props.lang) : []))
async function copy() {
  try { await navigator.clipboard.writeText(props.text); copied.value = true; setTimeout(() => (copied.value = false), 1800) } catch { /* sin permiso */ }
}
</script>

<template>
  <div class="cv">
    <div class="cv-bar">
      <span class="mono text-xs">{{ label || fileName }}<span v-if="lines.length" class="muted"> · {{ lines.length }} líneas</span></span>
      <span class="cv-actions">
        <button class="btn btn-sm" type="button" :disabled="!text" @click="copy"><span aria-live="polite">{{ copied ? 'Copiado' : 'Copiar' }}</span></button>
        <a v-if="downloadHref" class="btn btn-sm" :href="downloadHref" :download="fileName">Descargar</a>
      </span>
    </div>
    <div v-if="loading" class="cv-state" role="status" aria-busy="true"><div class="skeleton" style="height: 14px; width: 60%" /><div class="skeleton" style="height: 14px; width: 80%" /><div class="skeleton" style="height: 14px; width: 45%" /></div>
    <div v-else-if="error" class="cv-state"><div class="alert alert-error" role="alert">{{ error }}</div></div>
    <pre v-else tabindex="0" :aria-label="`Código ${label || fileName}`"><code><span v-for="(l, i) in lines" :key="i" class="ln"><span class="no" aria-hidden="true">{{ i + 1 }}</span><span class="tx" v-html="l || '&#8203;'" /></span></code></pre>
  </div>
</template>

<style scoped>
.cv { border: 1px solid var(--border-bright); border-radius: var(--radius-lg); overflow: hidden; background: #f6f8fb; }
.cv-bar { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 6px 10px 6px 14px; background: var(--bg-elevated); border-bottom: 1px solid var(--border); color: var(--text-secondary); flex-wrap: wrap; }
.muted { color: var(--text-muted); }
.cv-actions { display: flex; gap: 6px; }
.cv-state { padding: 18px; display: flex; flex-direction: column; gap: 10px; }
pre { margin: 0; padding: 12px 0; overflow: auto; max-height: 640px; font-family: var(--mono); font-size: 13px; line-height: 1.65; color: var(--text-primary); tab-size: 2; }
.ln { display: flex; min-width: max-content; }
.no { flex: 0 0 52px; text-align: right; padding-right: 14px; color: var(--text-muted); user-select: none; border-right: 1px solid var(--border); margin-right: 14px; }
.tx { white-space: pre; padding-right: 18px; }
.tx :deep(.tk-c) { color: #5a6a81; font-style: italic; }
.tx :deep(.tk-s) { color: #0a6a4b; }
.tx :deep(.tk-k) { color: #6639b0; font-weight: 700; }
.tx :deep(.tk-d) { color: #8f5300; }
.tx :deep(.tk-t) { color: #0b6f8c; }
.tx :deep(.tk-n) { color: #b42338; }
.tx :deep(.tk-p) { color: #08566d; }
</style>
