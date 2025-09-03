<script setup>
import { ref } from 'vue'
import { RouterLink } from 'vue-router'
import './auth.css'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import { getCSRFToken } from '@utils/session.js'
import { config } from '@config'

const spinnerRef = ref(false)

async function handleClick() {
  if (spinnerRef.value) return

  spinnerRef.value = true

  const response = await fetch(
    `${config.baseURL}/api/auth/payments/create-checkout-session`,
    {
      method: 'POST',
      headers: {
        'X-CSRF-Token': getCSRFToken(),
        Accept: 'application/json',
      },
      credentials: 'include',
    },
  )
  if (!response.ok) {
    if (response.status === 400) {
      console.error('User has an active subscription')
    } else {
      console.error('Error creating checkout session')
    }
    spinnerRef.value = false
    return
  } else {
    const data = await response.json()
    // eslint-disable-next-line no-undef
    const stripe = Stripe(config.stripeKey)
    stripe.redirectToCheckout({ sessionId: data.sessionId })
    spinnerRef.value = false
  }
}
</script>

<template>
  <div class="wrapper">
    <RouterLink target="_blank" class="docs-link" to="/documentation"
      ><h1>nodo.finance Professional</h1>
    </RouterLink>
    <button class="session-submit" @click="handleClick" :disabled="spinnerRef">
      <SpinnerWhite v-if="spinnerRef" />
      <span v-else>Go Professional</span>
    </button>
  </div>
</template>

<style scoped>
.wrapper {
  display: flex;
  flex-direction: column;
  margin: 0 auto;
  align-items: center;
  justify-content: center;
  gap: 25px;
}
h1 {
  border-bottom: none;
  text-align: center;
  padding-bottom: 0;
  margin: 0;
  letter-spacing: 0.5px;
}
.wrapper .session-submit {
  width: 100%;
  max-width: 100%;
}

/* media 320px */
@media (max-width: 320px) {
  .docs-link h1 {
    font-size: 21px;
  }
}
</style>
