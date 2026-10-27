<script setup>
import { ref } from 'vue'
const props = defineProps({ code: { type: String, required: true }, label: { type: String, default: '' } })
const copied = ref(false)
async function copy() {
  try { await navigator.clipboard.writeText(props.code); copied.value = true; setTimeout(() => (copied.value = false), 1800) } catch { /* sin permiso */ }
}
</script>

<template>
  <div class="cb">
    <div class="cb-bar">
      <span class="mono text-xs">{{ label }}</span>
      <button class="btn btn-sm" type="button" @click="copy"><span aria-live="polite">{{ copied ? 'Copiado' : 'Copiar' }}</span></button>
    </div>
    <pre tabindex="0"><code>{{ code }}</code></pre>
  </div>
</template>

<style scoped>
.cb { border: 1px solid var(--border-bright); border-radius: var(--radius-lg); overflow: hidden; background: #f6f8fb; }
.cb-bar { display: flex; justify-content: space-between; align-items: center; padding: 6px 10px 6px 14px; background: var(--bg-elevated); border-bottom: 1px solid var(--border); color: var(--text-secondary); }
pre { margin: 0; padding: 16px; overflow-x: auto; font-family: var(--mono); font-size: 13px; line-height: 1.6; color: var(--text-primary); tab-size: 2; }
</style>
