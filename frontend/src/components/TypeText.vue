<script setup>
import { computed } from 'vue'
const props = defineProps({ text: { type: String, default: '' }, known: { type: Object, default: () => new Set() } })
// Convierte los nombres completos (Chile.Comunes.Direccion) que existen en el catálogo en enlaces.
const parts = computed(() => {
  const out = []
  let last = 0
  for (const m of props.text.matchAll(/[A-Za-z_]\w*(?:\.[A-Za-z_]\w*)+/g)) {
    if (m.index > last) out.push({ t: props.text.slice(last, m.index) })
    out.push(props.known.has(m[0]) ? { t: m[0].split('.').pop(), id: m[0] } : { t: m[0].split('.').pop(), title: m[0] })
    last = m.index + m[0].length
  }
  if (last < props.text.length) out.push({ t: props.text.slice(last) })
  return out
})
</script>
<template><span class="mono ty"><template v-for="(p, i) in parts" :key="i"><RouterLink v-if="p.id" :to="`/esquemas/${p.id}`">{{ p.t }}</RouterLink><span v-else :title="p.title">{{ p.t }}</span></template></span></template>
<style scoped>.ty { font-size: 13px; }</style>
