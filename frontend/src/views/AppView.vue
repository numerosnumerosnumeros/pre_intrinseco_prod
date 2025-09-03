<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useHead } from '@unhead/vue'
import {
  appIsMountedStore,
  tickerSeachResultStore,
  tickerStore,
  fileNameStore,
  urlStore,
  periodStore,
  spinnerAppStore,
  parserWorkingStore,
  currencyStore,
  submitTypeStore,
} from '@state/app.js'
import { getCSRFToken } from '@utils/session.js'
import { config } from '@config'
import {
  validateCurrency,
  validatePeriod,
  validateTicker,
} from '@utils/sanitize.js'
import { extractPDFText } from '@utils/preprocess/pdf.js'
import { processHTMLText } from '@utils/preprocess/html.js'
import { Preprocessor } from '@utils/preprocess/preprocessor.js'
import LupaIcon from '@icons/LupaIcon.vue'
import RadioUnchecked from '@icons/RadioUnchecked.vue'
import RadioChecked from '@icons/RadioChecked.vue'
import CloseWhite from '@icons/CloseWhite.vue'
import FaviconWhiteSpinner from '@icons/FaviconWhiteSpinner.vue'
import PlusWhite from '@icons/PlusWhite.vue'
import FaviconWhite from '@icons/FaviconWhite.vue'
import FileIcon from '@icons/FileIcon.vue'
import SpinnerBlack from '@icons/SpinnerBlack.vue'
import LoadingWithinBox from '@icons/LoadingWithinBox.vue'
import AppTickers from '@components/app/AppTickers.vue'
import * as pdfjsLib from '@/lib/pdfjs/pdf.mjs'

const MAX_ITEMS = parseInt(import.meta.env.VITE_MAX_TICKERS)

// local stores
const errorAppRef = ref(false)
const errorLupaRef = ref(false)
const fileSpinnerRef = ref(false)
const fileNameRef = ref('')
const windowWidthRef = ref(null)
const showMobileMenuRef = ref(false)
const errorMsgRef = ref('')
const minPagRef = ref('')
const maxPagRef = ref('')

const updateWindowWidth = () => {
  window ? (windowWidthRef.value = window.innerWidth) : null
}

const setupPdfJs = () => {
  pdfjsLib.GlobalWorkerOptions.workerSrc = new URL(
    '@/lib/pdfjs/pdf.worker.mjs',
    import.meta.url,
  ).toString()
}

onMounted(async () => {
  updateWindowWidth()
  window.addEventListener('resize', updateWindowWidth)

  if (appIsMountedStore.value) {
    return
  }

  try {
    const resp = await fetch(`${config.baseURL}/api/app/mount`, {
      method: 'GET',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'X-CSRF-Token': getCSRFToken(),
      },
    })

    if (!resp.ok) throw new Error('Failed to mount tickers')

    const data = await resp.json()

    appIsMountedStore.value = data.tickers == null ? [] : data.tickers

    setupPdfJs()
  } catch (error) {
    console.error('Error mounting tickers:', error)
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', updateWindowWidth)
})

watch(
  () => tickerStore.value,
  newValue => {
    if (!newValue) {
      tickerSeachResultStore.value = null
      return
    }

    const sanitizedValue = newValue.replace(/[^A-Za-z0-9.]/g, '').toUpperCase()
    if (sanitizedValue !== newValue) {
      tickerStore.value = sanitizedValue
    }
  },
)

watch(
  () => periodStore.value,
  (newValue, oldValue) => {
    if (!newValue) {
      periodStore.value = ''
      return
    }

    if (oldValue && newValue.length < oldValue.length) {
      if (oldValue.endsWith('-') && newValue.length === 4) {
        periodStore.value = newValue.slice(0, 3)
        return
      }
    }

    let chars = newValue.toUpperCase().split('')

    // For first 4 positions, only allow numbers
    for (let i = 0; i < chars.length && i < 4; i++) {
      if (!/[0-9]/.test(chars[i])) {
        chars.splice(i, 1)
        i--
      }
    }

    // Position 4 must be '-'
    if (chars.length > 4 && chars[4] !== '-') {
      chars.splice(4)
    }

    // Position 5 must be S, Q or Y
    if (chars.length > 5 && !/[SQY]/.test(chars[5])) {
      chars.splice(5)
    }

    // Position 6 rules:
    if (chars.length > 6) {
      if (chars[5] === 'Y') {
        // No position 6 allowed for Y
        chars.splice(6)
      } else if (chars[5] === 'S') {
        // Only 1 or 2 allowed after S
        if (!/[1-2]/.test(chars[6])) {
          chars.splice(6)
        }
      } else if (chars[5] === 'Q') {
        // 1-4 allowed after Q
        if (!/[1-4]/.test(chars[6])) {
          chars.splice(6)
        }
      }
    }

    // Add hyphen after 4 digits if it doesn't exist
    if (chars.length === 4 && chars.every(char => /\d/.test(char))) {
      chars.push('-')
    }

    periodStore.value = chars.join('')
  },
)

function sanitizeNumber(pageNumber) {
  let sanitized = String(pageNumber).replace(/[^0-9]/g, '')
  sanitized = sanitized.replace(/^0+/, '')

  // Handle empty case
  if (sanitized === '') {
    return ''
  }

  // Enforce max length of 3
  if (sanitized.length > 3) {
    sanitized = sanitized.slice(0, 3)
  }

  return parseInt(sanitized)
}

watch(
  minPagRef,
  newValue => {
    const sanitized = sanitizeNumber(newValue)
    if (String(sanitized) !== String(newValue)) {
      minPagRef.value = sanitized
    }
  },
  { immediate: true },
)

watch(
  maxPagRef,
  newValue => {
    const sanitized = sanitizeNumber(newValue)
    if (String(sanitized) !== String(newValue)) {
      maxPagRef.value = sanitized
    }
  },
  { immediate: true },
)

function handleClickOutside() {
  if (spinnerAppStore.value) return
  showMobileMenuRef.value = false
  document.body.classList.remove('body-no-scroll')
}

function tooManyTickers(ticker) {
  if (appIsMountedStore.value === null) {
    return true
  }

  let tickerExists = appIsMountedStore.value.some(item => {
    if (item === ticker) {
      return true
    }
    return false
  })

  if (tickerExists) {
    return true
  }

  if (appIsMountedStore.value.length >= MAX_ITEMS) {
    return false
  }
  return true
}

async function submitFile() {
  if (windowWidthRef.value <= 680 && !showMobileMenuRef.value) {
    showMobileMenuRef.value = true
    document.body.classList.add('body-no-scroll')
    return
  }

  if (!appIsMountedStore.value) return
  if (spinnerAppStore.value) return
  if (parserWorkingStore.value) return
  if (errorAppRef.value) return

  const spaceForAnotherTicker = tooManyTickers(tickerStore.value)
  if (!spaceForAnotherTicker) {
    displayErrorApp('Too many tickers')
    return
  }

  if (!tickerStore.value || !validateTicker(tickerStore.value)) {
    displayErrorApp('Invalid ticker')
    return
  }

  if (!periodStore.value || !validatePeriod(periodStore.value)) {
    displayErrorApp('Invalid period')
    return
  }

  if (!currencyStore.value || !validateCurrency(currencyStore.value)) {
    displayErrorApp('Invalid currency')
    return
  }

  const submitReq = {
    ticker: tickerStore.value,
    period: periodStore.value,
    currency: currencyStore.value,
  }

  parserWorkingStore.value = 'Loading...'
  let content = null

  // *
  // **
  // ***
  // ****
  // ***** URL case
  if (submitTypeStore.value === 'url') {
    if (!urlStore.value) {
      displayErrorApp('Invalid URL')
      return
    }
    if (urlStore.value.length > 2048) {
      displayErrorApp('Invalid URL')
      return
    }

    const urlReq = {
      url: urlStore.value,
    }

    // Curl URL on server to avoid CORS issues
    spinnerAppStore.value = true

    try {
      const res = await fetch(`${config.baseURL}/api/app/url`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'X-CSRF-Token': getCSRFToken(),
        },
        body: JSON.stringify(urlReq),
      })

      if (res.ok) {
        const contentType = res.headers.get('Content-Type')
        if (contentType && contentType.includes('application/pdf')) {
          const blob = await res.blob()
          const file = new File([blob], 'document.pdf', {
            type: 'application/pdf',
          })
          content = await extractPDFText(pdfjsLib, file, [
            minPagRef.value,
            maxPagRef.value,
          ])
        } else {
          const text = await res.text()
          content = processHTMLText(text)
        }
      } else {
        throw new Error('Error fetching URL')
      }
    } catch (error) {
      console.error('Error fetching URL:', error)
      displayErrorApp('Error while fetching URL')
      cleanStates()
      return
    }
  }
  // **
  // ***
  // ****
  // ***** File case
  else {
    const fileInput = document.getElementById('fileUpload')
    if (!fileInput?.files?.length) {
      displayErrorApp('Select a file')
      return
    }

    fileNameStore.value = fileNameRef.value

    const file = fileInput.files[0]
    const fileType = file.type // MIME type

    if (fileType == 'application/pdf') {
      spinnerAppStore.value = true

      content = await extractPDFText(pdfjsLib, file, [
        minPagRef.value,
        maxPagRef.value,
      ])
    } else {
      const text = await file.text()
      content = processHTMLText(text)

      spinnerAppStore.value = true
    }
  }

  if (!content || content.length < 500) {
    displayErrorApp('Invalid file')
    cleanStates()
    return
  }

  const preprocessor = new Preprocessor()
  const preprocessedResult = preprocessor.preprocess(content, periodStore.value)
  // preprocessedResult = {
  //   balance_result: { chunk, cleaned, units, metrics, indicators },
  //   income_result: { chunk, cleaned, units, metrics, indicators },
  //   cash_flow_result: { chunk, cleaned, units, metrics, indicators },
  //   language: 'en' or 'es'
  // }

  if (
    preprocessedResult.balance_result.metrics.first_unique_hits < 17 &&
    preprocessedResult.income_result.metrics.first_unique_hits < 17 &&
    preprocessedResult.cash_flow_result.metrics.first_unique_hits < 17
  ) {
    displayErrorApp(
      'Invalid file. Is it a financial file? Is it in Spanish or English?',
    )
    cleanStates()
    return
  }

  const maxChunkSize = 14 * 1024 // 14KB
  if (
    preprocessedResult.balance_result.chunk.length > maxChunkSize ||
    preprocessedResult.income_result.chunk.length > maxChunkSize ||
    preprocessedResult.cash_flow_result.chunk.length > maxChunkSize
  ) {
    displayErrorApp('Invalid file')
    cleanStates()
    return
  }

  if (
    preprocessedResult.balance_result.cleaned.length > maxChunkSize ||
    preprocessedResult.income_result.cleaned.length > maxChunkSize ||
    preprocessedResult.cash_flow_result.cleaned.length > maxChunkSize
  ) {
    displayErrorApp('Invalid file')
    cleanStates()
    return
  }

  submitReq.content = preprocessedResult
  spinnerAppStore.value = true
  parserWorkingStore.value = null

  try {
    const res = await fetch(`${config.baseURL}/api/app/submit`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'X-CSRF-Token': getCSRFToken(),
      },
      body: JSON.stringify(submitReq),
    })

    if (!res.ok) {
      let errorMessage
      switch (res.status) {
        case 403:
          errorMessage = await res.text()
          break
        case 429: {
          errorMessage = 'Too many requests. Please try again later.'
          break
        }
        default: {
          try {
            const text = await res.text()
            if (text && text.startsWith('nodo error:')) {
              errorMessage = text.split(':')[1].trim()
            } else {
              errorMessage = text || ''
            }
          } catch {
            errorMessage = ''
          }
        }
      }
      throw new Error(errorMessage)
    }

    if (appIsMountedStore.value === null) {
      appIsMountedStore.value = []
    }

    let tickerIndex = -1
    if (appIsMountedStore.value.length > 0) {
      tickerIndex = appIsMountedStore.value.findIndex(
        item => item === tickerStore.value,
      )
    }

    if (tickerIndex !== -1) {
      appIsMountedStore.value.splice(tickerIndex, 1) // Remove existing ticker before unshifting it
    }

    appIsMountedStore.value.unshift(tickerStore.value)
    cleanStates()
  } catch (error) {
    displayErrorApp(error.message, 3000)
    fileNameStore.value = ''
  } finally {
    spinnerAppStore.value = false
    parserWorkingStore.value = null
  }
}

function updateFileName(event) {
  if (event.target instanceof HTMLInputElement && event.target.files) {
    const file = event.target.files[0]
    if (file) {
      fileNameRef.value = file.name
    } else {
      fileNameRef.value = ''
    }
  }
}

function displayErrorLupa() {
  tickerSeachResultStore.value = null
  tickerStore.value = ''
  errorLupaRef.value = true
  setTimeout(() => {
    errorLupaRef.value = false
  }, 500)
}

function displayErrorApp(message, timeout = 500) {
  spinnerAppStore.value = false
  parserWorkingStore.value = null
  errorMsgRef.value = message || 'Error occurred'
  errorAppRef.value = true
  setTimeout(() => {
    errorAppRef.value = false
    errorMsgRef.value = ''
  }, timeout)
}

function cleanStates() {
  errorAppRef.value = false
  errorLupaRef.value = false
  fileNameRef.value = ''
  fileNameStore.value = ''
  tickerStore.value = ''
  urlStore.value = ''
  periodStore.value = ''
  spinnerAppStore.value = false
  parserWorkingStore.value = null
  minPagRef.value = ''
  maxPagRef.value = ''
  cleanFileInput()
}

function cleanFileInput() {
  const fileInput = document.getElementById('fileUpload')
  if (fileInput) {
    fileInput.value = ''
    fileNameRef.value = ''
  }
}

function searchTicker() {
  if (
    !tickerStore.value ||
    tickerStore.value.length > 12 ||
    !validateTicker(tickerStore.value)
  ) {
    displayErrorLupa()
    return
  }

  try {
    const filteredTickers = appIsMountedStore.value
      .filter(ticker => {
        const tickerString = JSON.stringify(ticker)
        return tickerString.includes(tickerStore.value)
      })
      .slice(0, 20) // Limit to maximum 20 results

    if (filteredTickers.length === 0) {
      throw new Error('No results')
    } else {
      tickerSeachResultStore.value = filteredTickers
    }
  } catch {
    displayErrorLupa()
  }
}

function fileCountdown() {
  if (spinnerAppStore.value) return
  fileSpinnerRef.value = true
  setTimeout(() => {
    fileSpinnerRef.value = false
  }, 400)
}

useHead({
  title: 'nodo.finance - App',
  meta: [
    {
      name: 'twitter:title',
      content: 'nodo.finance - App',
    },
    {
      property: 'og:title',
      content: 'nodo.finance - App',
    },
    {
      name: 'robots',
      content: 'index, follow',
    },
  ],
})
</script>

<template>
  <div v-if="appIsMountedStore.value" class="whole-wrapper">
    <form
      @submit.prevent="submitFile()"
      class="filters-wrapper"
      :disabled="
        !appIsMountedStore.value ||
        spinnerAppStore.value ||
        errorAppRef ||
        parserWorkingStore.value
      "
    >
      <div class="search-box">
        <input
          placeholder="Company"
          class="ticker-text-input"
          type="text"
          maxlength="12"
          v-model="tickerStore.value"
          name="ticker"
          autocomplete="off"
          :disabled="
            !appIsMountedStore.value ||
            spinnerAppStore.value ||
            errorAppRef ||
            parserWorkingStore.value
          "
        />
        <button
          aria-label="Search"
          class="lupa"
          @click.stop.prevent="searchTicker"
          :disabled="
            !appIsMountedStore.value ||
            spinnerAppStore.value ||
            errorAppRef ||
            parserWorkingStore.value
          "
        >
          <CloseWhite v-if="errorLupaRef" />
          <LupaIcon v-else />
        </button>
      </div>

      <div class="filter-group" v-show="windowWidthRef > 680">
        <div class="radio-section">
          <div class="vertical">
            <button
              aria-label="File"
              @click.prevent.stop="
                () => {
                  submitTypeStore.value = 'file'
                  urlStore.value = ''
                }
              "
              class="radio-button"
              :disabled="
                submitTypeStore.value === 'file' ||
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                showMobileMenuRef ||
                parserWorkingStore.value
              "
            >
              <RadioChecked v-if="submitTypeStore.value === 'file'" />
              <RadioUnchecked v-else />
              &equiv;
            </button>
            <button
              aria-label="URL"
              @click.prevent.stop="
                () => {
                  submitTypeStore.value = 'url'
                  cleanFileInput()
                }
              "
              class="radio-button"
              :disabled="
                submitTypeStore.value === 'url' ||
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                showMobileMenuRef ||
                parserWorkingStore.value
              "
            >
              <RadioChecked v-if="submitTypeStore.value === 'url'" />
              <RadioUnchecked v-else />
              URL
            </button>
          </div>
        </div>
      </div>

      <!-- file -->
      <div
        class="horizontal"
        v-if="submitTypeStore.value === 'file'"
        v-show="windowWidthRef > 680"
      >
        <label
          for="fileUpload"
          class="custom-file-upload"
          @click="fileCountdown"
        >
          <i class="fa fa-cloud-upload" v-if="!fileNameRef"></i>
          <span v-if="fileNameStore.value" class="file-name-display">{{
            fileNameStore.value
          }}</span>
          <span v-else-if="fileNameRef" class="file-name-display">{{
            fileNameRef
          }}</span>
          <SpinnerBlack v-else-if="fileSpinnerRef" />
          <FileIcon v-else />
          <input
            id="fileUpload"
            type="file"
            accept="application/xhtml+xml, application/pdf, text/html, .mht, .mhtml"
            style="display: none"
            @change="updateFileName"
            autocomplete="off"
            :disabled="
              !appIsMountedStore.value ||
              spinnerAppStore.value ||
              errorAppRef ||
              parserWorkingStore.value
            "
          />
        </label>
        <div class="filter-group" v-show="windowWidthRef > 680">
          <div class="vertical">
            <input
              class="pag-text-input"
              type="text"
              v-model="minPagRef"
              placeholder="start"
              maxlength="3"
              name="period"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
            <input
              class="pag-text-input"
              type="text"
              v-model="maxPagRef"
              placeholder="end"
              maxlength="3"
              name="period"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
          </div>
        </div>
      </div>
      <!-- url -->
      <div
        class="horizontal"
        v-if="submitTypeStore.value === 'url'"
        v-show="windowWidthRef > 680"
      >
        <input
          placeholder="URL"
          class="url-text-input"
          type="text"
          v-model="urlStore.value"
          name="url"
          autocomplete="off"
          :disabled="
            !appIsMountedStore.value ||
            spinnerAppStore.value ||
            errorAppRef ||
            parserWorkingStore.value
          "
        />
        <div class="filter-group" v-show="windowWidthRef > 680">
          <div class="vertical">
            <input
              class="pag-text-input"
              type="text"
              v-model="minPagRef"
              placeholder="start"
              maxlength="3"
              name="period"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
            <input
              class="pag-text-input"
              type="text"
              v-model="maxPagRef"
              placeholder="end"
              maxlength="3"
              name="period"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
          </div>
        </div>
      </div>

      <input
        class="period-text-input"
        type="text"
        v-model="periodStore.value"
        placeholder="Period"
        maxlength="7"
        name="period"
        autocomplete="off"
        v-show="windowWidthRef > 680"
        :disabled="
          !appIsMountedStore.value ||
          spinnerAppStore.value ||
          errorAppRef ||
          parserWorkingStore.value
        "
      />

      <div class="filter-group" v-show="windowWidthRef > 680">
        <div class="radio-section">
          <div class="vertical">
            <button
              aria-label="Euro"
              @click.prevent.stop="() => (currencyStore.value = 'EUR')"
              class="radio-button"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            >
              <RadioChecked v-if="currencyStore.value === 'EUR'" />
              <RadioUnchecked v-else />
              €
            </button>
            <button
              aria-label="Dollar"
              @click.prevent.stop="() => (currencyStore.value = 'USD')"
              class="radio-button"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            >
              <RadioChecked v-if="currencyStore.value === 'USD'" />
              <RadioUnchecked v-else />
              $
            </button>
          </div>
          <div class="vertical">
            <button
              aria-label="Pound"
              @click.prevent.stop="() => (currencyStore.value = 'GBP')"
              class="radio-button"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            >
              <RadioChecked v-if="currencyStore.value === 'GBP'" />
              <RadioUnchecked v-else />
              £
            </button>
            <button
              aria-label="Not specified"
              @click.prevent.stop="() => (currencyStore.value = 'NA')"
              class="radio-button"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            >
              <RadioChecked v-if="currencyStore.value === 'NA'" />
              <RadioUnchecked v-else />
              -
            </button>
          </div>
        </div>
      </div>

      <div class="submit-button-container">
        <button
          aria-label="Send"
          class="finances"
          @click.prevent="submitFile"
          :disabled="
            !appIsMountedStore.value ||
            spinnerAppStore.value ||
            errorAppRef ||
            showMobileMenuRef ||
            parserWorkingStore.value
          "
        >
          <CloseWhite v-if="errorAppRef && !showMobileMenuRef" />
          <FaviconWhiteSpinner
            v-else-if="spinnerAppStore.value && !showMobileMenuRef"
          />
          <PlusWhite v-else-if="windowWidthRef <= 680" />
          <FaviconWhite v-else />
        </button>
        <p v-if="errorMsgRef" class="error-msg">{{ errorMsgRef }}</p>
        <p v-else-if="parserWorkingStore" class="parser-working-msg">
          {{ parserWorkingStore.value }}
        </p>
      </div>

      <!-- mobile menu -->
      <div
        class="mobile-wrapper"
        v-show="showMobileMenuRef && windowWidthRef <= 680"
      >
        <div class="backdrop" @click.prevent.stop="handleClickOutside"></div>
        <div class="mobile-menu-wrapper">
          <div class="filter-group">
            <div class="radio-section">
              <div class="vertical">
                <button
                  aria-label="File"
                  @click.prevent.stop="() => (submitTypeStore.value = 'file')"
                  class="radio-button"
                  :disabled="
                    submitTypeStore.value === 'file' ||
                    !appIsMountedStore.value ||
                    spinnerAppStore.value ||
                    errorAppRef ||
                    parserWorkingStore.value
                  "
                >
                  <RadioChecked v-if="submitTypeStore.value === 'file'" />
                  <RadioUnchecked v-else />
                  &equiv;
                </button>
                <button
                  aria-label="URL"
                  @click.prevent.stop="
                    () => {
                      submitTypeStore.value = 'url'
                      cleanFileInput()
                    }
                  "
                  class="radio-button"
                  :disabled="
                    submitTypeStore.value === 'url' ||
                    !appIsMountedStore.value ||
                    spinnerAppStore.value ||
                    errorAppRef ||
                    parserWorkingStore.value
                  "
                >
                  <RadioChecked v-if="submitTypeStore.value === 'url'" />
                  <RadioUnchecked v-else />
                  URL
                </button>
              </div>
            </div>
          </div>

          <!-- file -->
          <div v-if="submitTypeStore.value === 'file'">
            <label
              for="fileUpload"
              class="custom-file-upload"
              @click="fileCountdown"
            >
              <i class="fa fa-cloud-upload" v-if="!fileNameRef"></i>
              <span v-if="fileNameStore.value" class="file-name-display">{{
                fileNameStore.value
              }}</span>
              <span v-else-if="fileNameRef" class="file-name-display">{{
                fileNameRef
              }}</span>
              <SpinnerBlack v-else-if="fileSpinnerRef" />
              <FileIcon v-else />
              <input
                id="fileUpload"
                type="file"
                accept="application/xhtml+xml, application/pdf, text/html, .mht"
                style="display: none"
                @change="updateFileName"
                autocomplete="off"
                :disabled="
                  !appIsMountedStore.value ||
                  spinnerAppStore.value ||
                  errorAppRef ||
                  parserWorkingStore.value
                "
              />
            </label>
          </div>
          <!-- url -->
          <div v-if="submitTypeStore.value === 'url'">
            <input
              placeholder="URL"
              class="url-text-input"
              type="text"
              v-model="urlStore.value"
              name="url"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
          </div>

          <div class="vertical">
            <input
              class="pag-text-input-mobile"
              type="text"
              v-model="minPagRef"
              placeholder="start"
              maxlength="3"
              name="period"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
            <input
              class="pag-text-input-mobile"
              type="text"
              v-model="maxPagRef"
              placeholder="end"
              maxlength="3"
              name="period"
              autocomplete="off"
              :disabled="
                !appIsMountedStore.value ||
                spinnerAppStore.value ||
                errorAppRef ||
                parserWorkingStore.value
              "
            />
          </div>

          <input
            class="period-text-input"
            type="text"
            v-model="periodStore.value"
            placeholder="Period"
            maxlength="7"
            name="period"
            autocomplete="off"
            :disabled="
              !appIsMountedStore.value ||
              spinnerAppStore.value ||
              errorAppRef ||
              parserWorkingStore.value
            "
          />

          <div class="filter-group">
            <div class="radio-section">
              <div class="vertical">
                <button
                  aria-label="Euro"
                  @click.prevent.stop="() => (currencyStore.value = 'EUR')"
                  class="radio-button"
                  :disabled="
                    !appIsMountedStore.value ||
                    spinnerAppStore.value ||
                    errorAppRef ||
                    parserWorkingStore.value
                  "
                >
                  <RadioChecked v-if="currencyStore.value === 'EUR'" />
                  <RadioUnchecked v-else />
                  €
                </button>
                <button
                  aria-label="Dollar"
                  @click.prevent.stop="() => (currencyStore.value = 'USD')"
                  class="radio-button"
                  :disabled="
                    !appIsMountedStore.value ||
                    spinnerAppStore.value ||
                    errorAppRef ||
                    parserWorkingStore.value
                  "
                >
                  <RadioChecked v-if="currencyStore.value === 'USD'" />
                  <RadioUnchecked v-else />
                  $
                </button>
              </div>
              <div class="vertical">
                <button
                  aria-label="Pound"
                  @click.prevent.stop="() => (currencyStore.value = 'GBP')"
                  class="radio-button"
                  :disabled="
                    !appIsMountedStore.value ||
                    spinnerAppStore.value ||
                    errorAppRef ||
                    parserWorkingStore.value
                  "
                >
                  <RadioChecked v-if="currencyStore.value === 'GBP'" />
                  <RadioUnchecked v-else />
                  £
                </button>
                <button
                  aria-label="Not specified"
                  @click.prevent.stop="() => (currencyStore.value = 'NA')"
                  class="radio-button"
                  :disabled="
                    !appIsMountedStore.value ||
                    spinnerAppStore.value ||
                    errorAppRef ||
                    parserWorkingStore.value
                  "
                >
                  <RadioChecked v-if="currencyStore.value === 'NA'" />
                  <RadioUnchecked v-else />
                  -
                </button>
              </div>
            </div>
          </div>

          <button
            aria-label="Send"
            class="finances"
            @click.prevent="submitFile"
            :disabled="
              !appIsMountedStore.value ||
              spinnerAppStore.value ||
              errorAppRef ||
              parserWorkingStore.value
            "
          >
            <CloseWhite v-if="errorAppRef" />
            <FaviconWhiteSpinner v-else-if="spinnerAppStore.value" />
            <FaviconWhite v-else />
          </button>
        </div>
      </div>
    </form>

    <div class="content-wrapper">
      <div class="element2">
        <AppTickers />
      </div>
    </div>
  </div>
  <div v-else>
    <LoadingWithinBox />
  </div>
</template>

<style scoped>
.submit-button-container {
  position: relative;
}

.error-msg {
  position: absolute;
  top: 105%;
  left: 50%;
  transform: translateX(-50%);
  margin: 0;
  font-size: 9px;
  font-weight: 400;
  color: red;
  width: 140px;
  text-align: center;
  line-height: 1.2;
}

.parser-working-msg {
  position: absolute;
  top: 110%;
  left: 50%;
  transform: translateX(-50%);
  margin: 0;
  font-size: 9px;
  font-weight: 400;
  color: var(--gray-five);
  width: 140px;
  text-align: center;
  line-height: 1.2;
}

.vertical {
  display: flex;
  flex-direction: column;
  gap: 5px;
}
.divisa-p {
  width: 40px;
  text-align: right;
}
.filters-wrapper .lupa {
  background-color: black;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 42px;
  height: 40px;
  border: solid 2px black;
  border-radius: 10px;
  padding: 8px;
}

.whole-wrapper {
  margin: 0 auto;
  width: 680px;
  min-height: 550px;
}
.filters-wrapper {
  display: flex;
  justify-content: center;
  gap: 30px;
  align-items: center;
  padding: 10px 0;
  height: 40px;
  width: 100%;
  background-color: var(--gray-two);
  box-shadow: 0 0 10px var(--gray-two);
  border-radius: 20px;
}

input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
input[type='number'] {
  -moz-appearance: textfield;
  appearance: textfield;
}

.relative {
  position: relative;
}

.content-wrapper {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin: 30px auto 40px auto;
  gap: 50px;
}
.horizontal {
  display: flex;
  gap: 10px;
  align-items: center;
  height: 100%;
  max-height: 40px;
}

p,
span,
button.radio-button {
  margin: 0;
  font-size: 14px;
  letter-spacing: 1px;
}
.idioma-p {
  width: 35px;
}
input.ticker-text-input {
  display: flex;
  justify-content: center;
  outline: none;
  border: none;
  border-radius: 10px;
  box-sizing: border-box;
  background-color: var(--gray-one);
  width: 105px;
  height: 100%;
  max-height: 40px;
  margin: 0;
  padding: 0 5px;
  text-align: center;
  letter-spacing: 0.6px;
}
input.period-text-input {
  display: flex;
  justify-content: center;
  outline: none;
  border: none;
  border-radius: 10px;
  box-sizing: border-box;
  background-color: var(--gray-one);
  width: 75px;
  height: 100%;
  max-height: 40px;
  margin: 0;
  padding: 0;
  text-align: center;
  letter-spacing: 0.6px;
}

.filter-group {
  display: flex;
  gap: 5px;
}
.filter-group p {
  margin: 0;
  margin-right: 10px;
  text-align: right;
}
.radio-section {
  display: flex;
  gap: 10px;
}
.radio-button {
  display: flex;
  gap: 5px;
  align-items: center;
  background-color: transparent;
  border: none;
  font-size: 18px;
  padding: 0;
}

.pag-text-input {
  display: flex;
  justify-content: center;
  outline: none;
  border: none;
  border-radius: 5px;
  box-sizing: border-box;
  background-color: var(--gray-one);
  font-size: 10px;
  width: 40px;
  height: 18px;
  padding: 0 5px;
  margin: 0;
  text-align: center;
  letter-spacing: 0.6px;
}

.pag-text-input-mobile {
  display: flex;
  justify-content: center;
  outline: none;
  border: none;
  border-radius: 5px;
  box-sizing: border-box;
  background-color: var(--gray-one);
  font-size: 10px;
  max-width: 50px;
  height: 25px;
  padding: 0 5px;
  margin: 0;
  text-align: center;
  letter-spacing: 0.6px;
}

.element2 {
  height: 380px;
  width: 100%;
  border-radius: 30px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  background-color: var(--gray-one);
}

.url-text-input {
  display: flex;
  justify-content: center;
  outline: none;
  border: none;
  border-radius: 10px;
  box-sizing: border-box;
  background-color: var(--gray-one);
  width: 55px;
  height: 100%;
  height: 40px;
  padding: 0 5px;
  margin: 0;
  text-align: center;
}

.custom-file-upload {
  background-color: var(--gray-three);
  color: black;
  width: 55px;
  height: 100%;
  height: 40px;
  border: none;
  border-radius: 10px;
  box-sizing: border-box;
  display: flex;
  justify-content: center;
  align-items: center;
}
.custom-file-upload:hover {
  cursor: pointer;
}

.file-name-display {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  width: 45px;
}

button.finances {
  display: flex;
  justify-content: center;
  align-items: center;
  background-color: black;
  color: white;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  text-decoration: none;
  outline: none;
  letter-spacing: 1px;
  width: 42px;
  box-sizing: border-box;
  height: 40px;
  text-align: center;
  padding: 8px;
}

.loading {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 250px auto;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 15px;
  height: 100%;
  max-height: 40px;
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
.mobile-menu-wrapper {
  position: fixed;
  top: 0;
  right: 0;
  width: 120px;
  height: 100%;
  background-color: var(--gray-two);
  border-radius: 15px 0 0 15px;
  display: flex;
  flex-direction: column;
  padding: 20px;
  box-sizing: border-box;
  box-shadow: -2px 0 10px rgba(0, 0, 0, 0.3);
  z-index: 999;
  padding-top: 40px;
  gap: 25px;
  align-items: center;
}

.mobile-wrapper {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 100;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}

.mobile-menu-wrapper input {
  width: 80px;
}

@media (max-width: 760px) {
  .whole-wrapper {
    width: 650px;
  }
}
@media (max-width: 680px) {
  .whole-wrapper {
    width: 400px;
    margin: 10px auto 0 auto;
  }

  .content-wrapper {
    margin: 10px auto 0 auto;
  }

  .filters-wrapper {
    justify-content: center;
    padding: 10px 10px;
    width: fit-content;
    gap: 15px;
    margin: 0 auto;
  }
  .element2 {
    height: 100%;
    min-height: 200px;
    /* check with tickers filled -> */
    justify-content: center;
    margin-bottom: 20px;
  }
}
@media (max-width: 430px) {
  .whole-wrapper {
    width: 100%;
  }
  .element2 {
    font-size: 14px;
    padding: 0 10px;
    border-radius: 0;
    background-color: var(--gray-two);
    box-shadow: 0 0 10px var(--gray-two);
  }
}
@media (max-width: 310px) {
  .element2 {
    padding: 0;
  }
}

@media (min-width: 350px) and (max-height: 340px) {
  .mobile-menu-wrapper {
    padding: 5px 0;
    gap: 0;
    justify-content: space-around;
  }
}
</style>
