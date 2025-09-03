<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import {
  appIsMountedStore,
  tickerSeachResultStore,
  spinnerAppStore,
} from '@state/app.js'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import ArrowLeft from '@icons/ArrowLeft.vue'
import ArrowRight from '@icons/ArrowRight.vue'
import { RouterLink } from 'vue-router'

const pageCarteraRef = ref(1)
const loadingNextRef = ref(false)
const windowWidthRef = ref(null)

const updateWindowWidth = () => {
  window ? (windowWidthRef.value = window.innerWidth) : null
}

onMounted(async () => {
  updateWindowWidth()
  window.addEventListener('resize', updateWindowWidth)
})
onUnmounted(() => {
  window.removeEventListener('resize', updateWindowWidth)
})

const itemsPerPage = 20

const dataToDisplay = computed(() => {
  if (tickerSeachResultStore.value) {
    const startIndex = (pageCarteraRef.value - 1) * itemsPerPage
    return tickerSeachResultStore.value.slice(
      startIndex,
      startIndex + itemsPerPage,
    )
  }

  return (
    appIsMountedStore.value?.slice(
      (pageCarteraRef.value - 1) * itemsPerPage,
      pageCarteraRef.value * itemsPerPage,
    ) || []
  )
})

async function navPage(direction) {
  if (spinnerAppStore.value) return

  if (direction === 'prev' && pageCarteraRef.value > 1) {
    pageCarteraRef.value--
    return
  }

  if (tickerSeachResultStore.value) {
    return
  }

  // Handle next page with server-side pagination
  if (direction === 'next') {
    const currentItems = appIsMountedStore.value.length

    if (pageCarteraRef.value * itemsPerPage < currentItems) {
      // We have enough items locally, just move to next page
      pageCarteraRef.value++
    }
  }
}
</script>

<template>
  <div class="wrapper" v-if="dataToDisplay.length === 0">
    <p class="new-client">You haven't added any company yet</p>
  </div>
  <div class="wrapper" v-else>
    <div class="grid-container" v-if="windowWidthRef > 680">
      <div
        v-for="(item, index) in dataToDisplay"
        :key="index"
        class="grid-item"
        :class="{ disabled: spinnerAppStore.value }"
      >
        <RouterLink
          :to="{
            name: 'ticker',
            query: {
              ticker: item,
            },
          }"
          class="item-link"
        >
          <span class="span">{{ item }}</span>
        </RouterLink>
      </div>

      <!-- Render empty items to maintain the structure -->
      <div
        v-for="index in 20 - dataToDisplay.length"
        :key="'empty-' + index"
        class="grid-item empty"
      >
        <div class="item-link">
          <span class="span"></span>
        </div>
      </div>
    </div>
    <!-- mobile -->
    <div class="mobile-grid-container" v-else>
      <div
        v-for="(item, index) in dataToDisplay"
        :key="index"
        class="grid-item"
        :class="{ disabled: spinnerAppStore.value }"
      >
        <RouterLink
          :to="{
            name: 'ticker',
            query: {
              ticker: item,
            },
          }"
          class="item-link"
        >
          <span class="span">{{ item }}</span>
        </RouterLink>
      </div>

      <!-- Render empty items to maintain the structure -->
      <div
        v-for="index in 20 - dataToDisplay.length"
        :key="'empty-' + index"
        class="grid-item empty"
      >
        <div class="item-link">
          <span class="span"></span>
        </div>
      </div>
    </div>
  </div>
  <div class="navigation" v-if="dataToDisplay.length > 0">
    <button
      aria-label="Previous page"
      @click="navPage('prev')"
      :disabled="loadingNextRef"
    >
      <ArrowLeft />
    </button>
    <span>{{ pageCarteraRef }}</span>
    <button
      aria-label="Next page"
      @click="navPage('next')"
      :disabled="loadingNextRef"
    >
      <SpinnerWhite v-if="loadingNextRef" />
      <ArrowRight v-else />
    </button>
  </div>
</template>

<style scoped>
.disabled {
  pointer-events: none;
  opacity: 0.5;
}
.wrapper {
  display: flex;
  flex-direction: column;
  justify-content: space-around;
  height: 100%;
}

.grid-container {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-template-rows: repeat(6, auto);
  gap: 10px;
  margin: 0 20px;
}

.grid-item {
  display: flex;
  align-items: center;
  height: 40px;
  justify-content: center;
  letter-spacing: 1px;
}

.item-link {
  padding: 5px 10px;
}

.grid-item.empty {
  visibility: hidden;
}

a:hover {
  cursor: pointer;
  background-color: var(--gray-three);
  border-radius: 8px;
}

.navigation {
  display: flex;
  gap: 6px;
  align-items: center;
  justify-content: center;
  margin: 0 auto 30px auto;
  height: 40px;
}

.navigation button {
  border: none;
  border-radius: 10px;
  padding: 8px;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 40px;
  height: 40px;
  background-color: black;
}

.navigation span {
  width: 15px;
  font-size: 14px;
  text-align: center;
}
.new-client {
  font-size: 20px;
  text-align: center;
  letter-spacing: 1px;
  color: black;
}
.mobile-grid-container {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  grid-template-rows: repeat(12, auto);
  gap: 10px;
  margin: 0 10px;
  padding-top: 20px;
  box-sizing: border-box;
}
@media (max-width: 680px) {
  .new-client {
    font-size: 16px;
  }
}

@media (max-width: 500px) {
  .item-link {
    padding: 4px 6px;
  }
  .grid-item {
    height: 30px;
  }
  .new-client {
    padding: 0 15px;
  }
}
</style>
