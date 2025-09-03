<script setup>
import { ref, computed, watch } from 'vue'
import './auth.css'
import { emailInAuthFlowStore, authFlowStore } from '@state/session.js'
import {
  isValidEmail,
  isValidPassword,
  hasMinLength,
  hasLowerCase,
  hasUpperCase,
  hasNumber,
  hasSpecialChar,
} from '@utils/sanitize.js'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import { config } from '@config'

const showPasswordRef = ref(false)
const passwordRef = ref('')
const errorMsgRef = ref('')
const spinnerRef = ref(false)

function togglePasswordVisibility() {
  showPasswordRef.value = !showPasswordRef.value
}

const minLengthValid = computed(() => hasMinLength(passwordRef.value))
const upperCaseValid = computed(() => hasUpperCase(passwordRef.value))
const lowerCaseValid = computed(() => hasLowerCase(passwordRef.value))
const numberValid = computed(() => hasNumber(passwordRef.value))
const specialCharValid = computed(() => hasSpecialChar(passwordRef.value))

async function handleSubmit(event) {
  if (spinnerRef.value) return

  const formData = new FormData(event.target)
  const data = {
    email: emailInAuthFlowStore.value,
    password: formData.get('pwd'),
  }
  if (!isValidPassword(data.password)) {
    return
  }
  if (!isValidEmail(data.email)) {
    errorMsgRef.value = 'invalid email'
    return
  }

  spinnerRef.value = true

  try {
    const response = await fetch(`${config.baseURL}/api/auth/signup`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    if (!response.ok) throw new Error('error')

    authFlowStore.value = 'verify'
  } catch {
    errorMsgRef.value = 'verify your data'
    spinnerRef.value = false
  }
}

watch(
  () => errorMsgRef.value,
  newValue => {
    if (newValue) {
      setTimeout(() => {
        errorMsgRef.value = ''
      }, 1500)
    }
  },
)
</script>

<template>
  <form class="session-form" @submit.prevent="handleSubmit">
    <input
      class="email-input"
      type="email"
      id="email"
      name="email"
      v-model="emailInAuthFlowStore.value"
      readonly
      autocomplete="email"
    />
    <div class="password-input">
      <input
        type="text"
        placeholder="Enter your password"
        id="pwd"
        name="pwd"
        required
        autocomplete="current-password"
        v-model="passwordRef"
        v-if="showPasswordRef"
      />
      <input
        type="password"
        placeholder="Enter your password"
        id="pwd"
        name="pwd"
        required
        autocomplete="current-password"
        v-model="passwordRef"
        v-else
      />
      <button
        class="toggle-password"
        type="button"
        @click.prevent.stop="togglePasswordVisibility"
      >
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 576 512">
          <path
            fill="#C2C2C2"
            d="M288 32c-80.8 0-145.5 36.8-192.6 80.6C48.6 156 17.3 208 2.5 243.7c-3.3 7.9-3.3 16.7 0 24.6C17.3 304 48.6 356 95.4 399.4C142.5 443.2 207.2 480 288 480s145.5-36.8 192.6-80.6c46.8-43.5 78.1-95.4 93-131.1c3.3-7.9 3.3-16.7 0-24.6c-14.9-35.7-46.2-87.7-93-131.1C433.5 68.8 368.8 32 288 32zM144 256a144 144 0 1 1 288 0 144 144 0 1 1 -288 0zm144-64c0 35.3-28.7 64-64 64c-7.1 0-13.9-1.2-20.3-3.3c-5.5-1.8-11.9 1.6-11.7 7.4c.3 6.9 1.3 13.8 3.2 20.7c13.7 51.2 66.4 81.6 117.6 67.9s81.6-66.4 67.9-117.6c-11.1-41.5-47.8-69.4-88.6-71.1c-5.8-.2-9.2 6.1-7.4 11.7c2.1 6.4 3.3 13.2 3.3 20.3z"
          />
        </svg>
      </button>
    </div>
    <div class="session-cls-signup">
      <p class="red" v-if="errorMsgRef">{{ errorMsgRef }}</p>
      <div class="password-requirements" v-else>
        <p>
          <span :class="{ green: minLengthValid, red: !minLengthValid }"
            >Length,
          </span>
          <span :class="{ green: upperCaseValid, red: !upperCaseValid }"
            >uppercase,
          </span>
          <span :class="{ green: lowerCaseValid, red: !lowerCaseValid }"
            >lowercase,
          </span>
          <span :class="{ green: numberValid, red: !numberValid }"
            >number,
          </span>
          <span :class="{ green: specialCharValid, red: !specialCharValid }"
            >special</span
          >
        </p>
      </div>
    </div>
    <button
      class="session-submit"
      type="submit"
      name="submit"
      :disabled="spinnerRef"
    >
      <SpinnerWhite v-if="spinnerRef" />
      <span v-else>Sign up</span>
    </button>
  </form>
</template>
