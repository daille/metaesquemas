<script setup>
import Logo from './Logo.vue'
</script>

<template>
  <footer class="foot">
    <div class="container in">
      <div class="who">
        <Logo :size="28" />
        <p class="text-sm text-muted">Repositorio y catálogo de metaesquemas. Esquemas definidos una vez en TypeSpec y proyectados a JSON Schema, XSD, Protobuf y OpenAPI.</p>
      </div>
      <nav class="links" aria-label="Pie de página">
        <RouterLink to="/catalogo">Catálogo</RouterLink>
        <RouterLink to="/api">API</RouterLink>
      </nav>
    </div>
  </footer>
</template>

<style scoped>
.foot { border-top: 1px solid var(--border); background: var(--bg-surface); padding: 28px 0; }
.in { display: flex; justify-content: space-between; align-items: center; gap: 24px; flex-wrap: wrap; }
.who { display: flex; gap: 12px; align-items: center; max-width: 560px; }
.links { display: flex; gap: 20px; font-size: 14px; }
</style>
