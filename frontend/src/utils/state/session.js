import { reactive } from 'vue'

export const tokenStateStore = reactive({
  // Unmounted, MountedNotLoggedIn, MountedNonPremium, MountedPremium
  value: 'Unmounted',
})

export const authReadyPromise = {
  promise: null,
  resolve: null,
}

export const initAuthCheck = () => {
  authReadyPromise.promise = new Promise(resolve => {
    authReadyPromise.resolve = resolve
  })
}

export const setTokenMountedStore = premiumState => {
  tokenStateStore.value = premiumState

  if (authReadyPromise.resolve) {
    authReadyPromise.resolve()
  }
}

// tracks if the overlay for handling session is active and the current page of the auth flow
export const areWeHandlingSessionStore = reactive({
  value: false,
})
export const authFlowStore = reactive({
  value: 'waiting',
})
export const emailInAuthFlowStore = reactive({
  value: '',
})

export const resetHandlingSession = () => {
  areWeHandlingSessionStore.value = false
  authFlowStore.value = 'waiting'
  emailInAuthFlowStore.value = ''
  document.body.classList.remove('body-no-scroll')
}
