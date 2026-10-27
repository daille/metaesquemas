<script setup>
import { computed } from 'vue'
const props = defineProps({ tipo: String })
const map = { basico: ['Dato básico', 'badge-green'], estructurado: ['Estructurado', 'badge-accent'], documental: ['Documental', 'badge-purple'], servicio: ['Servicio', 'badge-amber'] }
const v = computed(() => map[props.tipo] || [props.tipo, 'badge-code'])
</script>
<template><span class="badge" :class="v[1]">{{ v[0] }}</span></template>
