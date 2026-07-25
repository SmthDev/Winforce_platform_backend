<script setup lang="ts">
defineProps<{
  active:  'profile'
}>()
</script>

<template>
  <header class="header">
    <a class="header__logo" href="/home">Winforce</a>

    <nav class="header__nav">
      <!-- <a class="header__link" :class="{ 'header__link--active': active === 'home' }" href="/home"> -->
        <!-- Home
      </a> -->
      <a
        class="header__link"
        :class="{ 'header__link--active': active === 'profile' }"
        href="/profile"
      >
        Profile
      </a>
    </nav>

    <a class="header__logout" href="/">Log out</a>
  </header>
</template>

<style scoped>
.header {
  display: flex;
  align-items: center;
  gap: 40px;
}

.header__logo {
  font-family: var(--wf-font-display);
  font-size: 22px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: var(--tracking-tight);
  text-decoration: none;
}

.header__nav {
  display: flex;
  align-items: center;
  gap: 28px;
  flex: 1 1 auto;
}

.header__link {
  color: var(--wf-label);
  font-size: 15px;
  font-weight: 700;
  text-decoration: none;
}

.header__link--active {
  color: var(--wf-text);
}

.header__logout {
  color: var(--wf-text);
  font-size: 15px;
  font-weight: 700;
  text-decoration: none;
}

@media (max-width: 600px) {
  .header {
    flex-wrap: wrap;
    gap: 16px 24px;
  }

  .header__nav {
    order: 3;
    flex: 1 0 100%;
  }
}
</style>
