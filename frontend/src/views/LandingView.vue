<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api/client'
import Icon from '../components/Icon.vue'
import TipoBadge from '../components/TipoBadge.vue'
import { TIPOS } from '../utils/labels'

const router = useRouter()
const q = ref('')
const stats = ref(null)
const error = ref('')
onMounted(async () => { try { stats.value = await api.stats() } catch (e) { error.value = e.message } })
const search = () => router.push({ path: '/catalogo', query: { q: q.value || undefined } })
const nf = new Intl.NumberFormat('es-CL')

const steps = [
  { n: '1', t: 'Se define una vez', d: 'Cada esquema se escribe en TypeSpec, con su responsable, tipo y vigencia declarados en el propio código.' },
  { n: '2', t: 'Se compila al cargar', d: 'Al subir el archivo, un worker valida el código y genera JSON Schema, XSD, Protobuf y OpenAPI. Nunca al consultar.' },
  { n: '3', t: 'Se consume en milisegundos', d: 'El portal y la API solo leen lo ya calculado, con todas las versiones y dependencias entre esquemas.' },
]
const access = [
  { to: '/catalogo', icon: 'layers', t: 'Explorar el catálogo', d: 'Busque y filtre por tipo, responsable, vigencia y uso.' },
  { to: '/catalogo?usada=si&sort=usos', icon: 'link', t: 'Esquemas más reutilizados', d: 'Qué elementos sostienen a más esquemas del Estado.' },
  { to: '/catalogo?tipo=servicio', icon: 'service', t: 'Contratos de servicios', d: 'APIs con su OpenAPI y servicios Protobuf.' },
  { to: '/api', icon: 'code', t: 'Integrar por API', d: 'Lea esquemas y proyecciones desde sus sistemas.' },
]
</script>

<template>
  <section class="hero">
    <div class="container hero-in">
      <div>
        <h1>Un esquema, todas las tecnologías</h1>
        <p class="lead">Repositorio central de los esquemas de datos del Estado: desde tipos básicos hasta contratos de servicios. Se definen una vez en TypeSpec y se publican proyectados a JSON Schema, XSD, Protobuf y OpenAPI.</p>
        <form class="qs" role="search" @submit.prevent="search">
          <label class="sr-only" for="q-home">Buscar un esquema</label>
          <input id="q-home" v-model="q" class="input" type="search" placeholder="Busque un esquema, namespace o responsable" autocomplete="off" />
          <button class="btn btn-primary" type="submit">Buscar</button>
        </form>
        <div class="cta">
          <RouterLink class="btn" to="/catalogo">Ver catálogo completo</RouterLink>
          <RouterLink class="btn" to="/api">Documentación de la API</RouterLink>
        </div>
      </div>
      <figure class="card code" aria-label="Ejemplo de un esquema TypeSpec">
        <p class="card-title">Ejemplo</p>
<pre><code><span class="c">/** Persona natural. */</span>
<span class="d">@responsable</span>(<span class="s">"División de Datos"</span>)
<span class="k">model</span> Persona {
  rut: <span class="t">Rut</span>;
  nombres: <span class="t">string</span>;
  domicilio?: <span class="t">Direccion</span>;
}</code></pre>
        <p class="proj"><span class="badge badge-accent">JSON Schema</span><span class="badge badge-accent">XSD</span><span class="badge badge-accent">Protobuf</span><span class="badge badge-accent">OpenAPI</span></p>
      </figure>
    </div>
  </section>

  <section class="container block" aria-labelledby="h-stats">
    <h2 id="h-stats" class="h2">El catálogo en cifras</h2>
    <div v-if="error" class="alert alert-error" role="alert">{{ error }}</div>
    <div class="stats" aria-live="polite">
      <RouterLink v-for="t in TIPOS" :key="t.id" :to="{ path: '/catalogo', query: { tipo: t.id } }" class="card stat">
        <div v-if="!stats" class="skeleton" style="height: 38px; width: 70px" aria-hidden="true" />
        <div v-else class="stat-value">{{ nf.format(stats.byTipo?.[t.id] ?? 0) }}</div>
        <div class="stat-label">{{ t.plural }}</div>
        <div class="text-xs text-muted">{{ t.hint }}</div>
      </RouterLink>
      <div class="card stat total">
        <div v-if="!stats" class="skeleton" style="height: 38px; width: 70px" aria-hidden="true" />
        <div v-else class="stat-value">{{ nf.format(stats.total) }}</div>
        <div class="stat-label">Esquemas en total</div>
        <div v-if="stats" class="text-xs text-muted">{{ stats.vigentes }} vigentes · {{ stats.responsables }} responsables</div>
      </div>
    </div>
  </section>

  <section v-if="stats?.masUsados?.length" class="container block" aria-labelledby="h-top">
    <h2 id="h-top" class="h2">Los más reutilizados</h2>
    <ol class="top card">
      <li v-for="e in stats.masUsados" :key="e.id">
        <RouterLink :to="`/esquemas/${e.id}`" class="nm">{{ e.name }}</RouterLink>
        <TipoBadge :tipo="e.tipo" />
        <span class="ns mono text-xs">{{ e.namespace }}</span>
        <span class="uses"><strong>{{ e.usedByCount }}</strong> {{ e.usedByCount === 1 ? 'esquema lo usa' : 'esquemas lo usan' }}</span>
      </li>
    </ol>
  </section>

  <section class="container block" aria-labelledby="h-how">
    <h2 id="h-how" class="h2">Cómo funciona</h2>
    <div class="steps">
      <article v-for="s in steps" :key="s.n" class="card step"><span class="num mono">{{ s.n }}</span><h3>{{ s.t }}</h3><p class="text-muted">{{ s.d }}</p></article>
    </div>
  </section>

  <section class="container block" aria-labelledby="h-acc">
    <h2 id="h-acc" class="h2">Acceso rápido</h2>
    <div class="access">
      <RouterLink v-for="a in access" :key="a.t" :to="a.to" class="card acc">
        <span class="ico"><Icon :name="a.icon" /></span><h3>{{ a.t }}</h3><p class="text-sm text-muted">{{ a.d }}</p>
      </RouterLink>
    </div>
  </section>
</template>

<style scoped>
.hero { background: var(--bg-surface); border-bottom: 1px solid var(--border); background-image: radial-gradient(circle, rgba(185, 197, 214, .55) 1px, transparent 1px); background-size: 30px 30px; }
.hero-in { display: grid; grid-template-columns: 1.1fr 1fr; gap: 48px; align-items: center; padding-top: 64px; padding-bottom: 64px; }
h1 { font-size: clamp(32px, 4.4vw, 50px); font-weight: 700; letter-spacing: -.03em; max-width: 14ch; }
.lead { font-size: 18px; color: var(--text-secondary); margin: 18px 0 26px; max-width: 54ch; }
.qs { display: flex; gap: 10px; max-width: 520px; }
.cta { display: flex; gap: 10px; margin-top: 22px; flex-wrap: wrap; }
.code { box-shadow: var(--shadow); background: rgba(255, 255, 255, .97); display: flex; flex-direction: column; gap: 12px; }
.code pre { font-family: var(--mono); font-size: 14px; line-height: 1.7; overflow-x: auto; }
.code .c { color: #5a6a81; font-style: italic; } .code .s { color: #0a6a4b; } .code .k { color: #6639b0; font-weight: 700; } .code .d { color: #8f5300; } .code .t { color: #0b6f8c; }
.proj { display: flex; gap: 6px; flex-wrap: wrap; border-top: 1px solid var(--border); padding-top: 12px; }
.block { padding-top: 56px; }
.h2 { font-size: 24px; letter-spacing: -.02em; margin-bottom: 20px; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(190px, 1fr)); gap: 16px; }
a.stat { color: var(--text-primary); display: block; }
a.stat:hover { text-decoration: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.total { background: var(--bg-elevated); }
.top { list-style: none; padding: 8px 20px; }
.top li { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 10px 0; border-bottom: 1px solid var(--border); }
.top li:last-child { border-bottom: 0; }
.nm { font-weight: 600; font-size: 16px; }
.ns { color: var(--text-secondary); }
.uses { margin-left: auto; color: var(--text-secondary); font-size: 14px; }
.steps { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 16px; }
.step h3, .acc h3 { font-size: 17px; margin: 10px 0 6px; }
.num { width: 34px; height: 34px; border-radius: 50%; background: var(--accent); color: #fff; display: inline-flex; align-items: center; justify-content: center; font-weight: 700; }
.access { display: grid; grid-template-columns: repeat(auto-fit, minmax(230px, 1fr)); gap: 16px; padding-bottom: 72px; }
.acc { color: var(--text-primary); }
.acc:hover { text-decoration: none; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-dim); }
.ico { width: 42px; height: 42px; border-radius: 10px; background: var(--accent-dim); color: var(--accent-strong); display: inline-flex; align-items: center; justify-content: center; }
@media (max-width: 900px) { .hero-in { grid-template-columns: 1fr; padding-top: 40px; padding-bottom: 40px; gap: 32px; } .uses { margin-left: 0; } }
</style>
