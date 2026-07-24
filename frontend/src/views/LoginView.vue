<script setup lang="ts">
import { computed, ref } from 'vue'
import PolyBackground from '../components/PolyBackground.vue'
import { login } from '../api'

const email = ref('')
const password = ref('')
const pending = ref(false)
const error = ref('')


const passwordRules = [
  {
    id: 'length',
    label: 'At least 8 characters',

    test: (value: string) => new TextEncoder().encode(value).length >= 8,
  },
  { id: 'uppercase', label: 'One uppercase letter', test: (value: string) => /[A-Z]/.test(value) },
  { id: 'number', label: 'One number', test: (value: string) => /[0-9]/.test(value) },
]

const passwordChecks = computed(() =>
  passwordRules.map((rule) => ({ id: rule.id, label: rule.label, ok: rule.test(password.value) })),
)

const passwordValid = computed(() => passwordChecks.value.every((check) => check.ok))


const showPasswordRules = computed(() => password.value.length > 0 && !passwordValid.value)

const canSubmit = computed(
  () => email.value.trim().length > 0 && passwordValid.value && !pending.value,
)

async function onSubmit() {
  if (!canSubmit.value) return

  pending.value = true
  error.value = ''
  try {
    await login(email.value.trim(), password.value)
    password.value = ''
    window.location.href = '/'
  } catch (e) {

    error.value =
      e instanceof TypeError
        ? 'Cannot reach the server. Please try again.'
        : e instanceof Error
          ? e.message
          : 'Something went wrong. Try again.'
  } finally {
    pending.value = false
  }
}
</script>

<template>
  <div class="login">
    <PolyBackground />

    <main class="login__main">
      <form class="login__form" novalidate @submit.prevent="onSubmit">
        <h1 class="login__title">Log in to Winforce Platform</h1>

        <div class="field">
          <label class="field__label" for="email">Email</label>
          <input
            id="email"
            v-model="email"
            class="field__input field__input--never-invalid"
            type="email"
            name="email"
            placeholder="your_email@gmail.com"
            autocomplete="email"
            inputmode="email"
            spellcheck="false"
            :disabled="pending"
            required
          />
        </div>

        <div class="field">
          <label class="field__label" for="password">Password</label>
          <input
            id="password"
            v-model="password"
            class="field__input"
            :class="{ 'field__input--invalid': showPasswordRules }"
            type="password"
            name="password"
            placeholder="••••••••"
            autocomplete="current-password"
            :aria-invalid="showPasswordRules"
            aria-describedby="password-rules"
            :disabled="pending"
            required
          />

          <ul
            v-if="showPasswordRules"
            id="password-rules"
            class="rules"
            aria-live="polite"
          >
            <li
              v-for="check in passwordChecks"
              :key="check.id"
              class="rules__item"
              :class="{ 'rules__item--ok': check.ok }"
            >
              <span class="rules__mark" aria-hidden="true">{{ check.ok ? '✓' : '·' }}</span>
              {{ check.label }}
            </li>
          </ul>
        </div>

        <p v-if="error" class="login__error" role="alert">{{ error }}</p>

        <button class="login__submit" type="submit" :disabled="!canSubmit">
          {{ pending ? 'Logging in…' : 'Log In' }}
        </button>
      </form>
    </main>
  </div>
</template>

<style scoped>
.login {
  position: relative;
  min-height: 100dvh;
  background-color: var(--wf-bg);
  overflow: hidden;
}



.login__main {
  position: relative;
  z-index: 1;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  /* Heavier bottom padding lifts the form above the true vertical centre, which
     reads better than dead-centre on a tall viewport. */
  padding: 48px 24px 176px;
}

.login__form {
  width: 100%;
  max-width: var(--wf-form-width);
}

.login__title {
  margin: 0 0 36px;
  font-family: var(--wf-font-display);
  font-size: clamp(30px, 4.4vw, 44px);
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: var(--tracking-tight);
  text-align: center;
  text-transform: lowercase;
}

.field + .field {
  margin-top: 20px;
}

.field__label {
  display: block;
  margin-bottom: 10px;
  color: var(--wf-label);
  font-size: 14px;
  font-weight: 700;
}

.field__input {
  width: 100%;
  height: var(--wf-control-height);
  padding: 0 18px;
  border: 1px solid var(--wf-field-border);
  border-radius: var(--wf-control-radius);
  background-color: var(--wf-field-bg);
  color: var(--wf-text);
  font-size: 16px;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease,
    box-shadow 0.15s ease;
}

.field__input::placeholder {
  color: var(--wf-placeholder);
}

.field__input:hover {
  border-color: var(--color-gray-800);
}

.field__input:focus {
  border-color: var(--wf-field-border-focus);
  background-color: rgba(255, 255, 255, 0.09);
  box-shadow: var(--wf-focus-ring);
  outline: none;
}

.field__input:focus-visible {
  outline: none;
}

.field__input--invalid,
.field__input--invalid:hover {
  border-color: var(--wf-field-border-invalid);
}

.field__input--invalid:focus {
  border-color: var(--wf-field-border-invalid);
  box-shadow: var(--wf-invalid-ring);
}

/* The email field is never flagged: a typo there is the server's call, not
   something to shout about while the user is still typing. Browsers style
   :invalid / :user-invalid on their own, so both are neutralised explicitly. */
.field__input--never-invalid:invalid,
.field__input--never-invalid:user-invalid {
  border-color: var(--wf-field-border);
  box-shadow: none;
}

.field__input--never-invalid:invalid:focus,
.field__input--never-invalid:user-invalid:focus {
  border-color: var(--wf-field-border-focus);
  box-shadow: var(--wf-focus-ring);
}

.field__input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.field__input:-webkit-autofill {
  -webkit-text-fill-color: var(--wf-text);
  -webkit-box-shadow: 0 0 0 1000px #121212 inset;
  caret-color: var(--wf-text);
}

/* The autofill rule replaces box-shadow wholesale, so the focus ring has to be
   re-declared alongside the inset fill. */
.field__input:-webkit-autofill:focus {
  -webkit-box-shadow:
    var(--wf-focus-ring),
    0 0 0 1000px #121212 inset;
}

.rules {
  margin: 10px 0 0;
  padding: 0;
  list-style: none;
}

.rules__item {
  display: flex;
  align-items: baseline;
  gap: 8px;
  color: var(--wf-label);
  font-family: var(--font-mono);
  font-size: 12px;
  line-height: 1.7;
  transition: color 0.15s ease;
}

.rules__item--ok {
  color: var(--wf-success);
}

.rules__mark {
  width: 10px;
  flex: 0 0 auto;
  text-align: center;
}

.login__error {
  margin: 18px 0 0;
  color: var(--wf-danger);
  font-size: 14px;
  font-weight: 700;
}

.login__submit {
  width: 100%;
  height: var(--wf-control-height);
  margin-top: 28px;
  border: none;
  border-radius: var(--wf-control-radius);
  background-color: var(--color-white);
  color: var(--color-black);
  font-size: 16px;
  font-weight: 700;


  text-transform: uppercase;
  letter-spacing: 0.06em;
  cursor: pointer;
  transition:
    background-color 0.15s ease,
    color 0.15s ease;
}

.login__submit:hover:not(:disabled) {
  background-color: var(--color-red);
  color: var(--color-white);
}

.login__submit:disabled {
  background-color: var(--color-gray-400);
  color: var(--color-black);
  cursor: not-allowed;
}


@media (max-width: 768px) {
  .login__main {
    align-items: flex-start;
    padding: clamp(64px, 18vh, 200px) 20px 48px;
  }

  .login__title {
    margin-bottom: 28px;
  }
}


@media (max-height: 640px) {
  .login__main {
    align-items: center;
    padding: 40px 24px 40px;
  }
}
</style>
