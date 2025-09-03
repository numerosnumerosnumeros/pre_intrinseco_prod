<script setup>
import { ref, watch, onUnmounted } from 'vue'
import ExpandIcon from '@icons/ExpandIcon.vue'
import CloseIcon from '@icons/CloseBlack.vue'
import SpinnerBlack from '@icons/SpinnerBlack.vue'

import MarkdownIt from 'markdown-it'

const showExpandedRef = ref(false)
const scrollPosition = ref(0)
const loadingTextRef = ref('')

const openExpanded = () => {
  scrollPosition.value = window.pageYOffset
  document.body.classList.add('body-no-scroll')
  document.body.style.top = `-${scrollPosition.value}px`
  showExpandedRef.value = true
}

const closeExpanded = () => {
  document.body.classList.remove('body-no-scroll')
  document.body.style.top = ''
  window.scrollTo(0, scrollPosition.value)
  showExpandedRef.value = false
}

const md = new MarkdownIt()

const { analyst, spinner } = defineProps({
  analyst: {
    type: String,
    required: false,
    default: null,
  },
  spinner: {
    type: Boolean,
    required: false,
    default: true,
  },
})

const displayedText = ref('')

const processText = text => {
  displayedText.value = md.render(text)
}

watch(
  () => analyst,
  newValue => {
    if (newValue) {
      processText(newValue)
    }
  },
  { immediate: true },
)

let timeoutIds = []

watch(
  () => spinner,
  newValue => {
    // Clear any existing timeouts
    timeoutIds.forEach(id => clearTimeout(id))
    timeoutIds = []

    if (newValue) {
      loadingTextRef.value = 'Loading data...'

      timeoutIds.push(
        setTimeout(() => {
          loadingTextRef.value = 'Analyzing company...'
        }, 5000),
      )

      timeoutIds.push(
        setTimeout(() => {
          loadingTextRef.value = 'Performing calculations...'
        }, 10000),
      )

      timeoutIds.push(
        setTimeout(() => {
          loadingTextRef.value = 'Generating analysis...'
        }, 15000),
      )
    } else {
      loadingTextRef.value = ''
    }
  },
  { immediate: true },
)

// Clean up timeouts when component is unmounted
onUnmounted(() => {
  timeoutIds.forEach(id => clearTimeout(id))
  closeExpanded()
})
</script>

<template>
  <div class="wrapper">
    <div v-if="spinner">
      <p class="mock">{{ loadingTextRef }}</p>
      <div class="loading-spinner">
        <SpinnerBlack />
      </div>
    </div>
    <div v-else-if="analyst" class="finances-wrapper">
      <p class="finances-content" v-html="displayedText"></p>
      <button
        aria-label="Expand"
        class="expand-icon"
        @click.prevent.stop="openExpanded"
      >
        <ExpandIcon />
      </button>
    </div>
    <div v-else>
      <p class="mock">You haven't analyzed this company yet</p>
      <p class="three-dots">...</p>
    </div>
  </div>

  <div class="expanded-wrapper" v-show="showExpandedRef">
    <div class="backdrop" @click.prevent.stop="closeExpanded"></div>
    <div class="expanded-content-wrapper">
      <div class="close-container">
        <button aria-label="Close" @click="closeExpanded" class="close-button">
          <CloseIcon />
        </button>
      </div>
      <p class="finances-expanded" v-html="displayedText"></p>
    </div>
  </div>
</template>

<style scoped>
.wrapper {
  padding: 20px;
  margin: 0;
  background-color: var(--gray-one);
  height: 100%;
  width: 100%;
  border-radius: 15px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  position: relative;
}

.three-dots {
  font-size: 30px;
  margin: 0;
  margin-top: 10px;
  color: var(--gray-five);
}

.finances-wrapper {
  position: relative;
  height: 100%;
}

.loading-spinner {
  margin: 40px auto;
}

.expand-icon {
  display: flex;
  justify-content: center;
  align-items: center;
  position: absolute;
  bottom: -3px;
  right: -3px;
  z-index: 2;
  background-color: var(--gray-three);
  border: none;
  border-radius: 10px;
  height: 40px;
  width: 40px;
}

.finances-content {
  white-space: pre-line;
  text-align: left;
  letter-spacing: 0.6px;
  margin-top: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 7;
  line-clamp: 7;
  -webkit-box-orient: vertical;
  white-space: normal;
}

.mock {
  color: var(--gray-five);
  letter-spacing: 0.6px;
}

.finances-content :deep(*) {
  font-size: 16px;
  margin: 0;
  margin-bottom: 10px;
  letter-spacing: 0.6px;
}
.finances-content :deep(h1) {
  font-size: 20px;
  font-weight: 600;
}
.finances-content :deep(h2) {
  font-size: 18px;
  font-weight: 600;
}
.finances-content :deep(ul) {
  list-style-type: disc;
  padding-left: 20px;
}

.backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.3);
  z-index: 200;
}

.expanded-wrapper {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 100;
  overflow: hidden;
}
.expanded-content-wrapper {
  position: fixed;
  top: 0;
  right: 0;
  width: 50vw;
  height: 100%;
  background-color: var(--gray-two);
  border-radius: 15px 0 0 0;
  display: flex;
  flex-direction: column;
  padding: 0;
  box-sizing: border-box;
  box-shadow: -2px 0 10px rgba(0, 0, 0, 0.3);
  z-index: 999;
  gap: 25px;
  align-items: center;
  overflow-y: auto;
}

.close-container {
  position: sticky;
  top: 0;
  width: 100%;
  z-index: 1000;
}

.close-button {
  position: absolute;
  top: 15px;
  left: 15px;
  background-color: var(--gray-three);
  border: none;
  border-radius: 10px;
  height: 40px;
  width: 40px;
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1001;
}

.finances-expanded {
  text-align: left;
  letter-spacing: 0.6px;
  margin: 0;
  padding: 25px 30px 60px 30px;
}

@media (max-width: 1050px) {
  .finances-content {
    -webkit-line-clamp: 22;
    line-clamp: 22;
  }
}

@media (max-width: 700px) {
  .expanded-content-wrapper {
    width: 100%;
    border-radius: 0;
  }

  .finances-expanded {
    padding: 20px 20px 60px 20px;
  }

  .finances-content {
    -webkit-line-clamp: 8;
    line-clamp: 8;
    margin-bottom: 50px;
  }
  .three-dots {
    margin-bottom: 40px;
  }
}
</style>
