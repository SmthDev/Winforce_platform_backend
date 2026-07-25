<script setup lang="ts">
import { computed } from 'vue'
import LoginView from './views/LoginView.vue'
import HomeView from './views/HomeView.vue'
import ProfileView from './views/ProfileView.vue'

const path = computed(() => window.location.pathname)
</script>

<template>
  <ProfileView v-if="path === '/profile'" />
  <HomeView v-else-if="path === '/home'" />
  <LoginView v-else />
</template>
