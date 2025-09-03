<script setup>
import { onBeforeMount, onUnmounted, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import FaviconLayout from '@icons/FaviconLayout.vue'
import LoginArrow from '@icons/LoginArrow.vue'
import LoginArrowLeft from '@icons/LoginArrowLeft.vue'
import LoginDoor from '@icons/LoginDoor.vue'
import SpinnerBlack from '@icons/SpinnerBlack.vue'
import { config } from '@config'
import {
  tokenStateStore,
  setTokenMountedStore,
  initAuthCheck,
  areWeHandlingSessionStore,
  authFlowStore,
} from '@state/session.js'
import AuthWrapper from '@components/auth/AuthWrapper.vue'
import { getCSRFToken } from '@utils/session.js'
import ProfileIcon from '@icons/ProfileIcon.vue'

const route = useRoute()
const windowWidth = ref(0)

function handleProfileClick() {
  document.body.classList.add('body-no-scroll')
  areWeHandlingSessionStore.value = true
  authFlowStore.value = 'profile'
}

const updateWindowWidth = () => {
  windowWidth.value = window.innerWidth
  window.addEventListener('resize', updateWindowWidth)
}

onBeforeMount(async () => {
  updateWindowWidth()

  if (tokenStateStore.value !== 'Unmounted') return

  initAuthCheck()

  try {
    const response = await fetch(`${config.baseURL}/api/auth/token`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'X-CSRF-Token': getCSRFToken(),
        Accept: 'application/json',
      },
    })

    if (!response.ok) throw new Error('error')

    const data = await response.json()
    if (data.plan) {
      setTokenMountedStore('MountedPremium')
    } else {
      setTokenMountedStore('MountedNonPremium')
    }
  } catch {
    setTokenMountedStore('MountedNotLoggedIn')
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', updateWindowWidth)
  document.body.classList.remove('body-no-scroll')
  areWeHandlingSessionStore.value = false
})
</script>

<template>
  <AuthWrapper v-if="areWeHandlingSessionStore.value" />
  <div class="top">
    <!-- LEFT -->
    <nav class="left-nav">
      <FaviconLayout />
      <!-- <h1 v-show="windowWidth > 850 && !route.path.startsWith('/app')">
        nodo.finance
      </h1> -->
    </nav>
    <!-- RIGHT -->
    <div class="right-nav">
      <button
        v-if="
          tokenStateStore.value !== 'Unmounted' &&
          tokenStateStore.value !== 'MountedNotLoggedIn'
        "
        v-show="!route.fullPath.includes('/app/company?ticker=')"
        class="small-button login"
        aria-label="Loading"
        @click.prevent="handleProfileClick"
        :disabled="tokenStateStore.value === 'Unmounted'"
      >
        <ProfileIcon />
      </button>
      <!-- arrow -->
      <button
        v-if="tokenStateStore.value === 'Unmounted'"
        v-show="!route.path.startsWith('/app')"
        class="small-button login"
        aria-label="Loading"
      >
        <SpinnerBlack />
      </button>
      <RouterLink
        v-else
        to="/app"
        class="small-button-router"
        v-show="!route.path.startsWith('/app')"
      >
        <LoginArrow v-if="tokenStateStore.value === 'MountedPremium'" />
        <LoginDoor v-else />
      </RouterLink>
    </div>
    <!-- WITHIN TICKER -->
    <div
      class="right-nav"
      v-show="route.fullPath.includes('/app/company?ticker=')"
    >
      <button
        v-if="tokenStateStore.value === 'Unmounted'"
        class="small-button login"
        aria-label="Loading"
      >
        <SpinnerBlack />
      </button>
      <RouterLink v-else to="/app" class="small-button-router">
        <LoginArrowLeft />
      </RouterLink>
    </div>
  </div>
</template>

<style scoped>
.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 95%;
  margin: 0 auto;
  height: 100px;
  padding-top: 23px;
  box-sizing: border-box;
}

.left-nav {
  display: flex;
  align-items: center;
  margin: 0;
  padding: 0;
  gap: 20px;
}

.right-nav {
  display: flex;
  align-items: center;
  margin: 0;
  padding: 0;
  gap: 20px;
}

a.nav-link {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  margin: 0 auto;
}

nav {
  display: flex;
  justify-content: center;
  align-items: center;
}

h1 {
  font-size: 22px;
  font-weight: 500;
  margin: 0;
  text-align: left;
  letter-spacing: 1px;
}
.login {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 38px;
  height: 38px;
  padding: 8px;
  border: none;
  border-radius: 10px;
}

.small-button-router {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background-color: var(--gray-one);
  text-decoration: none;
}

@media (max-width: 850px) {
  .top {
    height: 80px;
  }
  .left-nav {
    gap: 30px;
  }
}
</style>
