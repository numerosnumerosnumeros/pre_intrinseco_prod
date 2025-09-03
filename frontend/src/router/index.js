import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import {
  tokenStateStore,
  authReadyPromise,
  areWeHandlingSessionStore,
  authFlowStore,
} from '@state/session.js'

// route level code-splitting -> generates separate chunk (Name.[hash].js) for route, lazy-loaded when visited:
// component: () => import('../views/PageView.vue'),

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    // MAIN
    {
      path: '/',
      component: HomeView,
    },
    {
      path: '/health',
      redirect: '/',
    },
    {
      path: '/terms-of-service',
      component: () => import('../views/TermsView.vue'),
    },
    {
      path: '/documentation',
      component: () => import('../views/DocsView.vue'),
    },
    {
      path: '/welcome',
      component: () => import('../views/WelcomeView.vue'),
      meta: { requiresAuth: true },
    },

    // APP
    {
      path: '/app',
      component: () => import('../views/AppView.vue'),
      meta: { requiresPremium: true },
    },
    {
      path: '/app/company',
      name: 'ticker',
      component: () => import('../views/AppTickerView.vue'),
      meta: { requiresPremium: true },
    },

    // FALLBACK
    {
      path: '/:pathMatch(.*)*',
      redirect: '/',
    },
  ],
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    }
    return { top: 0 }
  },
})

router.beforeEach(async (to, from, next) => {
  // For direct navigation (first page load)
  if (tokenStateStore.value === 'Unmounted' && from.name === undefined) {
    // Wait for auth check to complete
    if (authReadyPromise.promise) {
      await authReadyPromise.promise
    }

    // Direct navigation: redirect to home if not allowed
    if (
      (to.meta.requiresPremium && tokenStateStore.value !== 'MountedPremium') ||
      (to.meta.requiresAuth &&
        !['MountedNonPremium', 'MountedPremium'].includes(
          tokenStateStore.value,
        ))
    ) {
      next('/')
      areWeHandlingSessionStore.value = true
      document.body.classList.add('body-no-scroll')
      if (
        to.meta.requiresPremium &&
        tokenStateStore.value === 'MountedNonPremium'
      ) {
        authFlowStore.value = 'wall'
      }
      return
    }
  }

  // For subsequent navigation
  else if (tokenStateStore.value === 'Unmounted' && from.name !== undefined) {
    next(false)
    return
  }

  // Requires premium
  if (to.meta.requiresPremium && tokenStateStore.value !== 'MountedPremium') {
    next(false)
    if (tokenStateStore.value === 'MountedNonPremium') {
      authFlowStore.value = 'wall'
    }
    areWeHandlingSessionStore.value = true
    document.body.classList.add('body-no-scroll')
    return
  }

  // Requires auth
  if (
    to.meta.requiresAuth &&
    !['MountedNonPremium', 'MountedPremium'].includes(tokenStateStore.value)
  ) {
    next(false)
    areWeHandlingSessionStore.value = true
    document.body.classList.add('body-no-scroll')
    return
  }

  next()
})

export default router
