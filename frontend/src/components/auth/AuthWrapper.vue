<script setup>
import { ref, watch } from 'vue'
import './auth.css'
import {
  emailInAuthFlowStore,
  authFlowStore,
  resetHandlingSession,
  tokenStateStore,
  setTokenMountedStore,
} from '@state/session.js'
import { isValidEmail } from '@utils/sanitize.js'
import CloseBlack from '@icons/CloseBlack.vue'
import AuthLogin from './AuthLogin.vue'
import AuthSignup from './AuthSignup.vue'
import AuthResend from './AuthResend.vue'
import AuthVerify from './AuthVerify.vue'
import AuthWall from './AuthWall.vue'
import AuthForgot from './AuthForgot.vue'
import AuthReset from './AuthReset.vue'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import { getCSRFToken } from '@utils/session.js'
import { config } from '@config'

const errorMsgRef = ref('')
const spinnerRef = ref(false)

watch(emailInAuthFlowStore, () => {
  emailInAuthFlowStore.value = emailInAuthFlowStore.value.toLowerCase()
})

watch(
  () => errorMsgRef.value,
  newValue => {
    if (newValue) {
      setTimeout(() => {
        errorMsgRef.value = ''
      }, 600)
    }
  },
)

async function startAuth(flow) {
  let emailValid = isValidEmail(emailInAuthFlowStore.value)
  if (!emailValid) {
    errorMsgRef.value = 'invalid email'
    return
  }

  if (flow === 'signin') {
    authFlowStore.value = 'signin'
  } else if (flow === 'signup') {
    authFlowStore.value = 'signup'
  }
}

async function handlePortal() {
  if (spinnerRef.value) return

  spinnerRef.value = true

  try {
    const response = await fetch(
      `${config.baseURL}/api/auth/payments/create-portal-session`,
      {
        method: 'POST',
        headers: {
          'X-CSRF-Token': getCSRFToken(),
          Accept: 'application/json',
        },
        credentials: 'include',
      },
    )

    if (!response.ok) throw new Error('error')

    const responseJson = await response.json()
    spinnerRef.value = false
    window.location.href = responseJson.url
  } catch {
    console.error('Error creating portal session')
    spinnerRef.value = false
  }
}

async function handleCheckout() {
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
      console.error('User already has an active subscription')
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

async function handleLogout() {
  const response = await fetch(`${config.baseURL}/api/auth/signout`, {
    method: 'POST',
    credentials: 'include',
  })

  if (!response.ok) {
    console.error('Error logging out')
    return
  }

  resetHandlingSession()
  setTokenMountedStore(false)
  window.location.href = '/'
}
</script>

<template>
  <div class="session-wrapper">
    <div class="backdrop" @click.prevent.stop="resetHandlingSession"></div>
    <div v-if="authFlowStore.value === 'waiting'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <div class="session-form">
        <input
          type="email"
          placeholder="Enter your email"
          name="email"
          id="email"
          autocomplete="email"
          v-model="emailInAuthFlowStore.value"
          novalidate
        />
        <div class="session-cls">
          <p v-if="errorMsgRef" class="red">{{ errorMsgRef }}</p>
        </div>
        <div class="two-buttons">
          <button
            class="session-submit no-mg"
            type="button"
            name="submit"
            @click.prevent.stop="startAuth('signin')"
          >
            Sign in
          </button>
          <button
            class="session-submit no-mg"
            type="button"
            name="submit"
            @click.prevent.stop="startAuth('signup')"
          >
            Sign up
          </button>
        </div>
      </div>
    </div>
    <div v-else-if="authFlowStore.value === 'signup'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthSignup />
    </div>

    <div
      v-else-if="authFlowStore.value === 'profile'"
      class="session-container"
    >
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <div class="profile-overlay">
        <button
          v-if="tokenStateStore.value === 'MountedPremium'"
          class="plan"
          @click.prevent.stop="handlePortal"
          :disabled="spinnerRef"
        >
          <SpinnerWhite v-if="spinnerRef" />
          <span v-else>Plan: Professional</span>
        </button>
        <button
          v-if="tokenStateStore.value === 'MountedNonPremium'"
          class="plan"
          @click.prevent="handleCheckout"
          :disabled="spinnerRef"
        >
          <SpinnerWhite v-if="spinnerRef" />
          <span v-else>Plan: Free</span>
        </button>
        <button class="logout" @click.prevent.stop="handleLogout">
          <span>Logout</span>
        </button>
      </div>
    </div>

    <div v-else-if="authFlowStore.value === 'resend'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthResend />
    </div>
    <div v-else-if="authFlowStore.value === 'verify'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthVerify />
    </div>
    <div v-else-if="authFlowStore.value === 'signin'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthLogin />
    </div>
    <div v-else-if="authFlowStore.value === 'wall'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthWall />
    </div>
    <div v-else-if="authFlowStore.value === 'forgot'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthForgot />
    </div>
    <div v-else-if="authFlowStore.value === 'reset'" class="session-container">
      <button
        class="close-auth-button"
        @click.prevent.stop="resetHandlingSession"
      >
        <CloseBlack />
      </button>
      <AuthReset />
    </div>
  </div>
</template>

<style scoped>
.session-wrapper {
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  position: fixed;
  top: 0;
  left: 0;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 50;
}
.backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 100;
}

.session-container {
  background-color: var(--gray-two);
  position: fixed;
  padding: 40px;
  border-radius: 15px;
  text-align: center;
  display: flex;
  flex-direction: column;
  width: 340px;
  z-index: 150;
  box-shadow: 0 0 50px rgba(0, 0, 0, 0.5);
  box-sizing: border-box;
}
.close-auth-button {
  position: absolute;
  top: 8px;
  right: 3px;
  background: none;
  border: none;
  cursor: pointer;
  z-index: 200;
}

.plan,
.logout {
  display: flex;
  margin: 0 auto;
  justify-content: center;
  align-items: center;
  width: 200px;
  height: 45px;
  margin: 10px auto;
  padding: 8px;
  background-color: black;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  text-decoration: none;
  font-size: 18px;
  outline: none;
  letter-spacing: 1px;
}

@media (max-width: 680px) {
  .session-container {
    width: 100%;
    height: 100%;
    box-shadow: none;
    justify-content: center;
    align-items: center;
    padding: 20px;
    border-radius: 0;
  }
  .close-auth-button {
    position: fixed;
    top: 25px;
    right: 15px;
  }
}
</style>
