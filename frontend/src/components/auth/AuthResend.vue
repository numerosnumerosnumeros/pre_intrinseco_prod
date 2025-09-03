<script setup>
import { ref, watchEffect } from 'vue'
import './auth.css'
import {
  emailInAuthFlowStore,
  authFlowStore,
  areWeHandlingSessionStore,
} from '@state/session.js'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import { config } from '@config'

const spinnerRef = ref(false)
const errorMsg = ref('')

watchEffect(() => {
  if (!emailInAuthFlowStore.value) {
    authFlowStore.value = ''
    areWeHandlingSessionStore.value = false
  }
})

async function handleNotVerified() {
  if (spinnerRef.value) return
  if (!emailInAuthFlowStore.value) return

  spinnerRef.value = true

  try {
    const response = await fetch(`${config.baseURL}/api/auth/resend`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ email: emailInAuthFlowStore.value }),
    })
    if (!response.ok) throw new Error('error')

    authFlowStore.value = 'verify'
  } catch {
    spinnerRef.value = false
    displayError()
  }
}

function displayError() {
  errorMsg.value = 'Error: Unverified email'
  setTimeout(() => {
    errorMsg.value = ''
  }, 1000)
}
</script>

<template>
  <form @submit.prevent.stop="handleNotVerified" class="session-form">
    <input
      class="email-input"
      type="email"
      id="email"
      name="email"
      v-model="emailInAuthFlowStore.value"
      readonly
    />
    <button class="session-submit" type="submit" :disabled="spinnerRef">
      <span v-if="errorMsg">{{ errorMsg }}</span>
      <SpinnerWhite v-else-if="spinnerRef" />
      <span v-else>Verify Email</span>
    </button>
  </form>
</template>
