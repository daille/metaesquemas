<script setup>
import { computed } from 'vue'
const props = defineProps({ estado: String, vigencia: Object, detail: Boolean })
const map = { vigente: ['Vigente', 'badge-green'], vencida: ['Vencida', 'badge-red'], futura: ['Vigencia futura', 'badge-amber'], 'sin-vigencia': ['Sin vigencia definida', 'badge-code'] }
const v = computed(() => map[props.estado] || ['—', 'badge-code'])
const range = computed(() => (props.vigencia ? `${props.vigencia.desde} → ${props.vigencia.hasta || 'sin término'}` : ''))
</script>
<template><span class="badge" :class="v[1]" :title="range">{{ v[0] }}<template v-if="detail && range"> · {{ range }}</template></span></template>
