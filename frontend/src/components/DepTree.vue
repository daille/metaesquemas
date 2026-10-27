<script setup>
import { ref, watch } from 'vue'
import { api } from '../api/client'
import DepTreeNode from './DepTreeNode.vue'

const props = defineProps({ schemaId: { type: String, required: true }, direction: { type: String, default: 'uses' } })
const depth = ref(3)
const tree = ref(null)
const loading = ref(false)
const error = ref('')
let ctrl

async function load() {
  ctrl?.abort(); ctrl = new AbortController()
  loading.value = true; error.value = ''
  try { tree.value = await api.tree(props.schemaId, props.direction, depth.value, { signal: ctrl.signal }) } catch (e) { if (e.name !== 'AbortError') error.value = e.message } finally { loading.value = false }
}
watch(() => [props.schemaId, props.direction, depth.value], load, { immediate: true })
</script>

<template>
  <div>
    <div class="bar">
      <label class="form-label" for="depth">Profundidad</label>
      <select id="depth" v-model.number="depth" class="input" style="width: auto">
        <option v-for="n in [1, 2, 3, 4, 5, 6]" :key="n" :value="n">{{ n }} {{ n === 1 ? 'nivel' : 'niveles' }}</option>
      </select>
    </div>
    <div v-if="error" class="alert alert-error" role="alert">{{ error }}</div>
    <div v-else-if="loading && !tree" class="skeleton" style="height: 120px" aria-busy="true" />
    <p v-else-if="tree && !tree.children?.length" class="empty">{{ direction === 'usedBy' ? 'Ningún otro esquema utiliza este elemento.' : 'Este esquema no depende de otros.' }}</p>
    <ul v-else-if="tree" role="tree" :aria-label="direction === 'usedBy' ? 'Esquemas que usan este elemento' : 'Dependencias del esquema'" class="tree">
      <DepTreeNode :node="tree" root />
    </ul>
  </div>
</template>

<style scoped>
.bar { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
.tree { list-style: none; }
</style>
