<script setup>
import { ref, watchEffect, watch } from 'vue'
import {
  emailInAuthFlowStore,
  authFlowStore,
  areWeHandlingSessionStore,
} from '@state/session.js'
import { isValidEmail, isValidNumber } from '@utils/sanitize.js'
import './auth.css'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import { config } from '@config'

const errorMsgRef = ref('')
const spinnerRef = ref(false)

watchEffect(() => {
  if (!emailInAuthFlowStore.value) {
    authFlowStore.value = ''
    areWeHandlingSessionStore.value = false
  }
})

async function handleVerify(event) {
  if (spinnerRef.value) return
  if (!emailInAuthFlowStore.value) return

  const formData = new FormData(event.target)
  const data = {
    email: emailInAuthFlowStore.value,
    confirmationCode: formData.get('code'),
  }
  if (!isValidEmail(data.email)) {
    errorMsgRef.value = 'invalid email'
    return
  }
  if (!isValidNumber(data.confirmationCode)) {
    errorMsgRef.value = 'invalid code'
    return
  }

  spinnerRef.value = true

  try {
    const response = await fetch(`${config.baseURL}/api/auth/verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })

    if (!response.ok) throw new Error('Response not ok')

    authFlowStore.value = 'signin'
  } catch {
    errorMsgRef.value = 'some data is incorrect'
  } finally {
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
  <form @submit.prevent.stop="handleVerify" class="session-form">
    <input
      type="text"
      placeholder="Verification Code"
      name="code"
      id="code"
      required
    />
    <div class="session-cls-mid">
      <p v-if="errorMsgRef" class="red">{{ errorMsgRef }}</p>
      <p v-else-if="emailInAuthFlowStore.value" class="session-p">
        {{ emailInAuthFlowStore.value }} check your email
      </p>
      <p v-else class="session-p">Check your email</p>
    </div>
    <button class="session-submit" type="submit" :disabled="spinnerRef">
      <SpinnerWhite v-if="spinnerRef" />
      <span v-else>Verify</span>
    </button>
  </form>
</template>
