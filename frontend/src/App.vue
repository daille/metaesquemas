<script setup>
import AppHeader from './components/AppHeader.vue'
import AppFooter from './components/AppFooter.vue'
</script>

<template>
  <a class="skip-link" href="#contenido">Saltar al contenido</a>
  <AppHeader />
  <main id="contenido" tabindex="-1"><RouterView /></main>
  <AppFooter />
</template>

<style>
#app { min-height: 100vh; display: flex; flex-direction: column; }
main { flex: 1; outline: none; }
</style>
