<script setup>
defineProps({ name: String, size: { type: Number, default: 22 } })
const paths = {
  search: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zm9 16-4-4',
  graph: 'M12 5v5m0 0-6 7m6-7 6 7M12 2.5a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5zM6 15a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5zm12 0a2.5 2.5 0 1 0 0 5 2.5 2.5 0 0 0 0-5z',
  book: 'M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2V5zm2 14h13M8 7h7',
  code: 'm8 8-5 4 5 4m8-8 5 4-5 4m-3-10-2 12',
  clock: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zm0 5v5l3 2',
  link: 'M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1',
  shield: 'M12 3 5 6v5c0 4.5 3 8.2 7 10 4-1.8 7-5.5 7-10V6l-7-3zm-3 9 2 2 4-4',
  open: 'M4 7h16M4 12h16M4 17h10',
  layers: 'm12 3 9 5-9 5-9-5 9-5zm-9 9 9 5 9-5M3 16l9 5 9-5',
  file: 'M7 3h7l5 5v13H7V3zm7 0v5h5M10 13h6M10 17h6',
  service: 'M5 5h14v5H5V5zm0 9h14v5H5v-5zM8 7.5h.01M8 16.5h.01',
  user: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8zm-7 9a7 7 0 0 1 14 0',
}
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false"><path :d="paths[name]" /></svg>
</template>
