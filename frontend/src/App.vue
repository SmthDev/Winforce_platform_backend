<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getHealth } from './api'

const data = ref<string>('loading...')

onMounted(async () => {
  try {
    data.value = JSON.stringify(await getHealth(), null, 2)
  } catch (e) {
    data.value = String(e)
  }
})
</script>

<template>
  <pre>{{ data }}</pre>
</template>
