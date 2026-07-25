<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import PolyBackground from '../components/PolyBackground.vue'
import AppHeader from '../components/AppHeader.vue'
import { getProfile, updateProfileName } from '../api'

interface Stat {
  value: string
  label: string
}

const email = ref('')
const firstName = ref('')
const lastName = ref('')
const loading = ref(true)

const savingName = ref(false)
const nameError = ref('')

const displayName = computed(() => {
  const name = `${firstName.value} ${lastName.value}`.trim()
  return name || email.value
})

const stats: Stat[] = [
  { value: '84', label: 'Games' },
  { value: '19', label: 'Goals' },
]

const avatarUrl = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

onMounted(async () => {
  try {
    const profile = await getProfile()
    email.value = profile.email
    firstName.value = profile.first_name
    lastName.value = profile.last_name
  } catch {
    window.location.href = '/'
  } finally {
    loading.value = false
  }
})

async function onSaveName() {
  const first = firstName.value.trim()
  const last = lastName.value.trim()
  if (!first || !last) {
    nameError.value = 'Enter both first and last name.'
    return
  }

  savingName.value = true
  nameError.value = ''
  try {
    const profile = await updateProfileName(first, last)
    firstName.value = profile.first_name
    lastName.value = profile.last_name
  } catch (e) {
    nameError.value = e instanceof Error ? e.message : 'Failed to save. Try again.'
  } finally {
    savingName.value = false
  }
}

function browseFiles() {
  fileInput.value?.click()
}

function onFileChange(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  avatarUrl.value = URL.createObjectURL(file)
}
</script>

<template>
  <div class="profile">
    <PolyBackground />

    <div class="profile__content">
      <AppHeader active="profile" />

      <h1 class="profile__title">Profile</h1>

      <section class="card">
        <div class="card__avatar">
          <button type="button" class="avatar" @click="browseFiles">
            <img v-if="avatarUrl" :src="avatarUrl" alt="" class="avatar__image" />
            <template v-else>
              <svg
                class="avatar__icon"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                aria-hidden="true"
              >
                <rect x="3" y="3" width="18" height="18" rx="2" />
                <circle cx="9" cy="9" r="1.75" />
                <path d="m21 15-5-5L5 21" />
              </svg>
              <span class="avatar__label">Avatar</span>
            </template>
          </button>
          <span class="avatar__hint">
            or <a href="#" @click.prevent="browseFiles">browse files</a>
          </span>
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            class="avatar__input"
            @change="onFileChange"
          />
        </div>

        <div class="card__divider" aria-hidden="true"></div>

        <div class="card__body">
          <h2 class="identity__name">{{ loading ? 'Loading…' : displayName }}</h2>

          <form class="name-form" novalidate @submit.prevent="onSaveName">
            <div class="name-form__row">
              <div class="name-form__field">
                <label class="name-form__label" for="first-name">First name</label>
                <input
                  id="first-name"
                  v-model="firstName"
                  class="name-form__input"
                  type="text"
                  autocomplete="given-name"
                  :disabled="loading || savingName"
                />
              </div>
              <div class="name-form__field">
                <label class="name-form__label" for="last-name">Last name</label>
                <input
                  id="last-name"
                  v-model="lastName"
                  class="name-form__input"
                  type="text"
                  autocomplete="family-name"
                  :disabled="loading || savingName"
                />
              </div>
              <button class="name-form__submit" type="submit" :disabled="loading || savingName">
                {{ savingName ? 'Saving…' : 'Save' }}
              </button>
            </div>
            <p v-if="nameError" class="name-form__error" role="alert">{{ nameError }}</p>
          </form>

          <div class="rule" aria-hidden="true"></div>

          <dl class="stats">
            <div v-for="stat in stats" :key="stat.label" class="stats__item">
              <dt class="stats__value">{{ stat.value }}</dt>
              <dd class="stats__label">{{ stat.label }}</dd>
            </div>
          </dl>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.profile {
  position: relative;
  min-height: 100dvh;
  background-color: var(--wf-bg);
  overflow: hidden;
}

.profile__content {
  position: relative;
  z-index: 1;
  max-width: 1180px;
  margin: 0 auto;
  padding: 40px 40px 120px;
}

.profile__title {
  margin: 64px 0 40px;
  font-family: var(--wf-font-display);
  font-size: clamp(36px, 5vw, 56px);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: var(--tracking-tight);
}

.card {
  display: flex;
  align-items: stretch;
  gap: 56px;
  border: 1px solid var(--color-gray-900);
  border-radius: var(--radius-lg);
  background-color: rgba(0, 0, 0, 0.4);
  padding: 48px 56px;
}

.card__avatar {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex: 0 0 auto;
}

.avatar {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 148px;
  height: 148px;
  border: none;
  border-radius: var(--radius-pill);
  background-color: rgba(255, 255, 255, 0.04);
  color: var(--wf-label);
  cursor: pointer;
  overflow: hidden;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.avatar:hover {
  background-color: rgba(255, 255, 255, 0.08);
  color: var(--wf-text);
}

.avatar__image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar__icon {
  width: 28px;
  height: 28px;
}

.avatar__label {
  font-size: 14px;
}

.avatar__hint {
  margin-top: 14px;
  color: var(--wf-label);
  font-size: 14px;
}

.avatar__hint a {
  color: var(--wf-text);
  text-decoration: underline;
}

.avatar__input {
  display: none;
}

.card__divider {
  flex: 0 0 auto;
  width: 1px;
  background-color: var(--color-gray-900);
}

.card__body {
  flex: 1 1 auto;
  min-width: 0;
}

.identity__name {
  margin: 0;
  font-family: var(--wf-font-display);
  font-size: clamp(20px, 2.2vw, 28px);
  font-weight: 700;
  letter-spacing: var(--tracking-tight);
  word-break: break-word;
}

.name-form {
  margin-top: 20px;
}

.name-form__row {
  display: flex;
  align-items: flex-end;
  gap: 16px;
}

.name-form__field {
  flex: 1 1 200px;
}

.name-form__label {
  display: block;
  margin-bottom: 8px;
  color: var(--wf-label);
  font-size: 13px;
  font-weight: 700;
}

.name-form__input {
  width: 100%;
  height: 48px;
  padding: 0 16px;
  border: 1px solid var(--wf-field-border);
  border-radius: var(--wf-control-radius);
  background-color: var(--wf-field-bg);
  color: var(--wf-text);
  font-size: 15px;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease;
}

.name-form__input:focus {
  border-color: var(--wf-field-border-focus);
  background-color: rgba(255, 255, 255, 0.09);
  outline: none;
}

.name-form__input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.name-form__submit {
  flex: 0 0 auto;
  height: 48px;
  padding: 0 24px;
  border: none;
  border-radius: var(--wf-control-radius);
  background-color: var(--color-white);
  color: var(--color-black);
  font-size: 14px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.name-form__submit:hover:not(:disabled) {
  background-color: var(--color-red);
  color: var(--color-white);
}

.name-form__submit:disabled {
  background-color: var(--color-gray-400);
  color: var(--color-black);
  cursor: not-allowed;
}

.name-form__error {
  margin: 10px 0 0;
  color: var(--wf-danger);
  font-size: 13px;
  font-weight: 700;
}

.rule {
  margin: 32px 0;
  border-top: 1px solid var(--color-gray-900);
}

.stats {
  display: flex;
  gap: 64px;
  margin: 0;
}

.stats__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.stats__value {
  font-family: var(--wf-font-display);
  font-size: clamp(28px, 3vw, 36px);
  font-weight: 700;
  color: var(--color-white);
}

.stats__label {
  margin: 0;
  color: var(--wf-label);
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

@media (max-width: 768px) {
  .profile__content {
    padding: 28px 20px 80px;
  }

  .profile__title {
    margin: 40px 0 28px;
  }

  .card {
    flex-direction: column;
    align-items: center;
    gap: 32px;
    padding: 32px 24px;
  }

  .card__divider {
    display: none;
  }

  .card__body {
    width: 100%;
    text-align: center;
  }

  .name-form__row {
    flex-direction: column;
    align-items: stretch;
  }

  .name-form__submit {
    width: 100%;
  }

  .rule {
    margin: 24px 0;
  }

  .stats {
    justify-content: center;
    gap: 40px;
    flex-wrap: wrap;
  }
}
</style>
