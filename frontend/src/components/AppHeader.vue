<script setup>
import { ref } from 'vue'
import Logo from './Logo.vue'

const open = ref(false)
const links = [
  { to: '/', label: 'Inicio', exact: true },
  { to: '/catalogo', label: 'Catálogo de esquemas' },
  { to: '/api', label: 'API y ayuda' },
]
</script>

<template>
  <header class="site-header">
    <div class="container bar">
      <RouterLink to="/" class="brand" aria-label="Catálogo de Metaesquemas, ir al inicio" @click="open = false">
        <Logo :size="38" />
        <span class="brand-text">
          <span class="brand-name">Catálogo de Metaesquemas</span>
          <span class="brand-sub">TypeSpec · JSON Schema · XSD · Protobuf · OpenAPI</span>
        </span>
      </RouterLink>
      <nav id="menu-principal" class="nav" :class="{ open }" aria-label="Principal">
        <RouterLink v-for="l in links" :key="l.to" :to="l.to" class="nav-item" :exact-active-class="l.exact ? 'active' : ''"
          :active-class="l.exact ? '' : 'active'" @click="open = false">{{ l.label }}</RouterLink>
      </nav>
      <button class="menu-btn" type="button" :aria-expanded="open" aria-controls="menu-principal" @click="open = !open">
        <span class="sr-only">Menú</span>
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path v-if="!open" d="M4 7h16M4 12h16M4 17h16" /><path v-else d="M6 6l12 12M18 6 6 18" /></svg>
      </button>
    </div>
    <div class="glow-bar" aria-hidden="true" />
  </header>
</template>

<style scoped>
.site-header { position: sticky; top: 0; z-index: 50; background: rgba(255, 255, 255, .96); backdrop-filter: blur(8px); border-bottom: 1px solid var(--border); }
.bar { display: flex; align-items: center; gap: 24px; height: var(--header-h); }
.brand { display: flex; align-items: center; gap: 12px; color: var(--text-primary); }
.brand:hover { text-decoration: none; }
.brand-text { display: flex; flex-direction: column; line-height: 1.15; }
.brand-name { font-weight: 700; font-size: 17px; letter-spacing: -.01em; }
.brand-sub { font-family: var(--mono); font-size: 10px; color: var(--text-secondary); }
.nav { display: flex; gap: 4px; margin-left: 8px; flex: 1; }
.nav-item { padding: 8px 14px; border-radius: var(--radius); color: var(--text-secondary); font-weight: 500; font-size: 15px; }
.nav-item:hover { background: var(--bg-elevated); color: var(--text-primary); text-decoration: none; }
.nav-item.active { background: var(--accent-dim); color: var(--accent-strong); }
.menu-btn { display: none; margin-left: auto; background: none; border: 1px solid var(--border-bright); border-radius: var(--radius); padding: 7px; cursor: pointer; }
@media (max-width: 900px) {
  .brand-sub { display: none; }
  .menu-btn { display: inline-flex; }
  .nav { display: none; position: absolute; top: var(--header-h); left: 0; right: 0; flex-direction: column; background: var(--bg-surface); border-bottom: 1px solid var(--border); padding: 12px 16px; margin: 0; }
  .nav.open { display: flex; }
}
</style>
