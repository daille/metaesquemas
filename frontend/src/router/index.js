import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/', name: 'home', component: () => import('../views/LandingView.vue'), meta: { title: 'Inicio' } },
  { path: '/catalogo', name: 'catalog', component: () => import('../views/CatalogView.vue'), meta: { title: 'Catálogo de esquemas' } },
  { path: '/esquemas/:id', name: 'schema', component: () => import('../views/SchemaView.vue'), props: true, meta: { title: 'Esquema' } },
  { path: '/api', name: 'api', component: () => import('../views/ApiDocsView.vue'), meta: { title: 'API' } },
  { path: '/:pathMatch(.*)*', name: 'notfound', component: () => import('../views/NotFoundView.vue'), meta: { title: 'No encontrado' } },
]

export function createAppRouter() {
  const router = createRouter({
    history: createWebHistory(),
    routes,
    scrollBehavior: (to, from, saved) => saved || (to.path === from.path ? false : { top: 0 }),
  })
  router.afterEach((to) => { document.title = `${to.meta.title ? to.meta.title + ' · ' : ''}Catálogo de Metaesquemas` })
  return router
}
