<script setup>
import { ref, watch } from 'vue'
import './auth.css'
import { emailInAuthFlowStore, authFlowStore } from '@state/session.js'
import { isValidEmail } from '@utils/sanitize.js'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import { config } from '@config'

const errorMsgRef = ref('')
const spinnerRef = ref(false)

async function handleForgot(event) {
  if (spinnerRef.value) return

  const formData = new FormData(event.target)
  const data = {
    email: formData.get('email').trim().toLowerCase(),
  }

  if (!isValidEmail(data.email)) {
    errorMsgRef.value = 'invalid email'
    return
  }

  spinnerRef.value = true

  try {
    const response = await fetch(`${config.baseURL}/api/auth/forgot`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    if (!response.ok) throw new Error('Response not ok')
    authFlowStore.value = 'reset'
  } catch {
    errorMsgRef.value = 'error, please check your data'
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
  <form class="session-form" @submit.prevent.stop="handleForgot">
    <input
      type="email"
      id="email"
      name="email"
      v-model="emailInAuthFlowStore.value"
      readonly
      autocomplete="email"
    />
    <div class="session-cls-tall">
      <p v-if="errorMsgRef" class="red">{{ errorMsgRef }}</p>
      <p v-else class="session-p">
        If {{ emailInAuthFlowStore.value }} is a registered user, they will
        receive an email with a code.
      </p>
    </div>
    <button
      class="session-submit"
      type="submit"
      name="submit"
      :disabled="spinnerRef"
    >
      <SpinnerWhite v-if="spinnerRef" />
      <span v-else>Confirm</span>
    </button>
  </form>
</template>
