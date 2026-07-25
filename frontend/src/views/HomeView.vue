<script setup lang="ts">
import PolyBackground from '../components/PolyBackground.vue'
import AppHeader from '../components/AppHeader.vue'
</script>

<template>
  <div class="home">
    <PolyBackground />

    <div class="home__content">
      <AppHeader active="home" />

      <!-- <h1 class="home__title">Home</h1> -->
<!-- 
      <section class="card">
        <p class="card__text">You're logged in. Head over to your profile to see your stats.</p>
        <a class="card__cta" href="/profile">Go to Profile</a>
      </section> -->
    </div>
  </div>
</template>

<style scoped>
.home {
  position: relative;
  min-height: 100dvh;
  background-color: var(--wf-bg);
  overflow: hidden;
}

.home__content {
  position: relative;
  z-index: 1;
  max-width: 1180px;
  margin: 0 auto;
  padding: 40px 40px 120px;
}

.home__title {
  margin: 64px 0 40px;
  font-family: var(--wf-font-display);
  font-size: clamp(36px, 5vw, 56px);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: var(--tracking-tight);
}

.card {
  border: 1px solid var(--color-gray-900);
  border-radius: var(--radius-lg);
  background-color: rgba(0, 0, 0, 0.4);
  padding: 48px 56px;
}

.card__text {
  margin: 0 0 24px;
  color: var(--wf-label);
  font-size: 16px;
}

.card__cta {
  display: inline-flex;
  height: var(--wf-control-height);
  padding: 0 28px;
  align-items: center;
  justify-content: center;
  border-radius: var(--wf-control-radius);
  background-color: var(--color-white);
  color: var(--color-black);
  font-size: 15px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.card__cta:hover {
  background-color: var(--color-red);
  color: var(--color-white);
}

@media (max-width: 768px) {
  .home__content {
    padding: 28px 20px 80px;
  }

  .home__title {
    margin: 40px 0 28px;
  }

  .card {
    padding: 32px 24px;
  }
}
</style>
