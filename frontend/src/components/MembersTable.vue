<script setup>
import { computed } from 'vue'
import TypeText from './TypeText.vue'

const props = defineProps({ d: { type: Object, required: true } })
const known = computed(() => new Set((props.d.dependencies || []).filter((x) => x.exists).map((x) => x.id)))
const cons = (c) => (c ? Object.entries(c).map(([k, v]) => `${k}: ${v}`).join(' · ') : '')
</script>

<template>
  <div class="mt">
    <p v-if="d.extends" class="text-sm">Extiende a <TypeText :text="d.extends" :known="known" /></p>
    <p v-if="d.baseType" class="text-sm">Tipo base: <TypeText :text="d.baseType" :known="known" /> <span v-if="d.builtin && d.builtin !== d.baseType" class="text-muted">(escalar estándar: <span class="mono">{{ d.builtin }}</span>)</span></p>
    <p v-if="d.constraints && Object.keys(d.constraints).length" class="text-sm">Restricciones: <span class="mono">{{ cons(d.constraints) }}</span></p>
    <p v-if="d.plantilla" class="alert alert-info text-sm">Plantilla genérica ({{ (d.plantilla.parametros || []).join(', ') }}): no se proyecta a XSD ni Protobuf hasta que se instancia con tipos concretos.</p>

    <div v-if="d.members?.length" class="table-wrap">
      <table>
        <caption class="sr-only">Propiedades</caption>
        <thead><tr><th>Propiedad</th><th>Tipo</th><th>Obligatoria</th><th>Descripción y restricciones</th></tr></thead>
        <tbody>
          <tr v-for="m in d.members" :key="m.name">
            <td class="mono">{{ m.name }}</td><td><TypeText :text="m.type" :known="known" /></td>
            <td>{{ m.optional ? 'No' : 'Sí' }}</td>
            <td>{{ m.doc }}<span v-if="m.constraints" class="mono text-xs cs"> {{ cons(m.constraints) }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-if="d.values?.length" class="table-wrap">
      <table>
        <caption class="sr-only">Valores de la enumeración</caption>
        <thead><tr><th>Valor</th><th>Descripción</th></tr></thead>
        <tbody><tr v-for="v in d.values" :key="v.name"><td class="mono">{{ v.value }}</td><td>{{ v.doc }}</td></tr></tbody>
      </table>
    </div>
    <div v-if="d.variants?.length" class="table-wrap">
      <table>
        <caption class="sr-only">Variantes de la unión</caption>
        <thead><tr><th>Variante</th><th>Tipo</th></tr></thead>
        <tbody><tr v-for="(v, i) in d.variants" :key="i"><td class="mono">{{ v.name || '—' }}</td><td><TypeText :text="v.type" :known="known" /></td></tr></tbody>
      </table>
    </div>
    <div v-if="d.operations?.length" class="table-wrap">
      <table>
        <caption class="sr-only">Operaciones del servicio</caption>
        <thead><tr><th>Operación</th><th>Parámetros</th><th>Retorna</th></tr></thead>
        <tbody>
          <tr v-for="o in d.operations" :key="o.name">
            <td><strong class="mono">{{ o.name }}</strong><div class="text-sm text-muted">{{ o.doc }}</div></td>
            <td><span v-if="!o.params.length" class="text-muted">—</span><div v-for="p in o.params" :key="p.name"><span class="mono text-xs">{{ p.name }}{{ p.optional ? '?' : '' }}:</span> <TypeText :text="p.type" :known="known" /></div></td>
            <td><TypeText :text="o.returns" :known="known" /></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.mt { display: flex; flex-direction: column; gap: 14px; }
.cs { color: var(--text-secondary); display: block; }
</style>
