<script setup>
defineProps({ size: { type: Number, default: 40 } })
</script>

<template>
  <svg :width="size" :height="size" viewBox="0 0 64 64" aria-hidden="true" focusable="false">
    <rect width="64" height="64" rx="14" fill="var(--accent)" />
    <g fill="none" stroke="#fff" stroke-width="4" stroke-linecap="round" stroke-linejoin="round">
      <path d="M24 14c-5 0-7 2-7 7v5c0 3-2 5-5 6 3 1 5 3 5 6v5c0 5 2 7 7 7" />
      <path d="M40 14c5 0 7 2 7 7v5c0 3 2 5 5 6-3 1-5 3-5 6v5c0 5-2 7-7 7" />
    </g>
    <circle cx="32" cy="32" r="4" fill="#fff" />
  </svg>
</template>
