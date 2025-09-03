<script setup>
import { ref, onBeforeMount } from 'vue'
import { tokenStateStore, setTokenMountedStore } from '@state/session.js'
import { getCSRFToken } from '@utils/session.js'
import { config } from '@config'
import { useRoute } from 'vue-router'
import SpinnerBlack from '@icons/SpinnerBlack.vue'
import FaviconLayout from '@icons/FaviconLayout.vue'
import { useHead } from '@unhead/vue'

useHead({
  title: 'Verifying...',
  meta: [
    {
      name: 'twitter:title',
      content: '-',
    },
    {
      property: 'og:title',
      content: '-',
    },
    {
      name: 'robots',
      content: 'noindex',
    },
  ],
})

const updatedCookiesRef = ref(null)

const route = useRoute()
const tokenSlug = route.query.token

onBeforeMount(async () => {
  let attempts = 0
  while (tokenStateStore.value === 'Unmounted' && attempts < 20) {
    await new Promise(resolve => setTimeout(resolve, 500))
    attempts++
  }

  if (tokenStateStore.value === 'Unmounted') {
    updatedCookiesRef.value = 'error'
    return
  }

  if (tokenStateStore.value === 'MountedNotLoggedIn') {
    window.location.href = '/'
    return
  }

  await new Promise(resolve => setTimeout(resolve, 5000))

  try {
    const refreshResponse = await fetch(
      `${config.baseURL}/api/auth/payments/refresh-status?token=${tokenSlug}`,
      {
        method: 'POST',
        credentials: 'include',
        headers: {
          'X-CSRF-Token': getCSRFToken(),
        },
      },
    )
    if (!refreshResponse.ok) {
      if (refreshResponse.status !== 401) {
        updatedCookiesRef.value = 'error'
        return
      } else {
        throw new Error('error')
      }
    }

    const tokenResponse = await fetch(`${config.baseURL}/api/auth/token`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'X-CSRF-Token': getCSRFToken(),
        Accept: 'application/json',
        'Cache-Control': 'no-cache',
        Pragma: 'no-cache',
      },
    })

    if (tokenResponse.ok) {
      const data = await tokenResponse.json()
      if (data.plan) {
        setTokenMountedStore('MountedPremium')
      } else {
        setTokenMountedStore('MountedNonPremium')
      }
    } else {
      setTokenMountedStore('MountedNotLoggedIn')
    }

    await new Promise(resolve => setTimeout(resolve, 2000))
    window.location.href = '/'
  } catch {
    await new Promise(resolve => setTimeout(resolve, 2000))
    window.location.href = '/'
    return
  }
})
</script>

<template>
  <div class="wrapper loading" v-if="!updatedCookiesRef">
    <div class="John">
      <SpinnerBlack />
    </div>
    <h1>Verifying user</h1>
    <p>Please wait</p>
  </div>
  <div class="wrapper error" v-if="updatedCookiesRef === 'error'">
    <h1>An error has occurred</h1>
    <p>
      Log out and log back in. If the error persists, you can contact us at
      {{ config.contactEmail }}.
    </p>
    <FaviconLayout class="favicon" />
  </div>
</template>

<style scoped>
.wrapper {
  display: flex;
  flex-direction: column;
  text-align: center;
  position: fixed;
  gap: 50px;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  z-index: 1000;
  background-color: var(--gray-two);
  justify-content: center;
  align-items: center;
}
.wrapper h1 {
  margin: 0;
  padding: 0;
  font-size: 22px;
  font-weight: 500;
  letter-spacing: 1px;
}
.wrapper p {
  margin: 0;
  padding: 0;
  font-size: 16px;
  letter-spacing: 0.8px;
}

.loading p {
  padding-bottom: 50px;
}

.John {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

@media (max-width: 600px) {
  .wrapper p {
    width: 75%;
  }
  .wrapper h1 {
    font-size: 20px;
  }
}
</style>
