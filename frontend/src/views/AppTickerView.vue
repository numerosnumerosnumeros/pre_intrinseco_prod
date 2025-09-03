<script setup>
import { ref, onMounted, onUnmounted, watchEffect, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useHead } from '@unhead/vue'
import {
  appIsMountedStore,
  editAppStore,
  cursorTickerStore,
  tickerSeachResultStore,
  tickerStore,
} from '@state/app.js'
import { getCSRFToken } from '@utils/session.js'
import {
  calculateRatiosWithPrice,
  editableFields,
  calculateRatios,
  wishedPER,
  revenueNeededForWishedPER,
} from '@utils/ratios.js'
import {
  validateTicker,
  validateCurrency,
  filterNumericValues,
} from '@utils/sanitize.js'
import CloseWhite from '@icons/CloseWhite.vue'
import SpinnerWhite from '@icons/SpinnerWhite.vue'
import FaviconWhite from '@icons/FaviconWhite.vue'
import FaviconWhiteSpinner from '@icons/FaviconWhiteSpinner.vue'
import LoadingWithinBox from '@icons/LoadingWithinBox.vue'
import ArrowLeft from '@icons/ArrowLeft.vue'
import ArrowRight from '@icons/ArrowRight.vue'
import OkIcon from '@icons/OkIcon.vue'
import TrashIcon from '@icons/TrashIcon.vue'
import EditIcon from '@icons/EditIcon.vue'
import { config } from '@config'
import AppRatios from '@components/app/AppRatios.vue'
import AppEdit from '@components/app/AppEdit.vue'
import AppSankey from '@components/app/AppSankey.vue'
import AppAnalyst from '@components/app/AppAnalyst.vue'

const route = useRoute()
const router = useRouter()

const ticker = route.query.ticker

const tickerIsMounted = ref(null)
const dataToPassRef = ref(null)
const periodRef = ref('')
const customPriceRef = ref(null)
const customPERRef = ref(null)
const navSpinnerRef = ref(false)
const spinnerSaveEditRef = ref(false)
const editFlowRef = ref(false)
const spinnerDeleteRef = ref(false)
const isDeleteDropdownOpenRef = ref(false)
const windowWidthRef = ref(null)
const spinnerAnalystRef = ref(false)
const errorMsgRef = ref('')
const errorAppRef = ref(false)

const updateWindowWidth = () => {
  window ? (windowWidthRef.value = window.innerWidth) : null
}

watchEffect(() => {
  if (
    periodRef.value &&
    tickerIsMounted.value?.financial_data?.[periodRef.value]
  ) {
    dataToPassRef.value = tickerIsMounted.value.financial_data[periodRef.value]
    const ratios = calculateRatiosWithPrice(
      customPriceRef.value,
      tickerIsMounted.value.financial_data[periodRef.value],
    )
    Object.keys(ratios).forEach(key => {
      dataToPassRef.value[key] = ratios[key]
    })
  }
})

const dataToEditComputed = computed(() => {
  if (!dataToPassRef.value) return {}
  return Object.keys(dataToPassRef.value).reduce((acc, key) => {
    if (editableFields.includes(key)) {
      acc[key] = dataToPassRef.value[key]
    }
    return acc
  }, {})
})

onMounted(async () => {
  if (tickerIsMounted.value) return
  updateWindowWidth()
  window.addEventListener('resize', updateWindowWidth)
  await fetchTickerData()
})

async function fetchTickerData() {
  try {
    let url = new URL(`${config.baseURL}/api/app/mount-ticker?ticker=${ticker}`)
    if (cursorTickerStore.value) {
      url = new URL(
        `${config.baseURL}/api/app/mount-ticker?ticker=${ticker}&cursor=${cursorTickerStore.value}`,
      )
    }

    const res = await fetch(url.toString(), {
      method: 'GET',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'X-CSRF-Token': getCSRFToken(),
      },
    })

    if (!res.ok) throw new Error('Failed to fetch')

    const data = await res.json()

    const calculatedData = calculateRatios(data.financial_data)

    if (!tickerIsMounted.value) {
      tickerIsMounted.value = {
        currency: data.currency,
        financial_data: {
          [data.period]: calculatedData,
        },
        ...(data.analysis && { analyst: data.analysis }),
      }
    } else {
      tickerIsMounted.value.financial_data[data.period] = calculatedData
    }
    periodRef.value = data.period

    if (data.cursor) {
      cursorTickerStore.value = data.cursor
    } else {
      cursorTickerStore.value = null
    }
  } catch (error) {
    console.error('Failed to fetch:', error)
    router.push('/app')
  }
}

onUnmounted(() => {
  window.removeEventListener('resize', updateWindowWidth)
  cursorTickerStore.value = null
})

function editButton() {
  if (navSpinnerRef.value) return
  editFlowRef.value = true
  editAppStore.value = JSON.parse(JSON.stringify(dataToEditComputed.value))
}

function cancelEditButton() {
  if (spinnerSaveEditRef.value) return
  editAppStore.value = Object.keys(dataToPassRef.value).reduce((acc, key) => {
    if (editableFields.includes(key)) {
      acc[key] = dataToPassRef.value[key]
    }
    return acc
  }, {})

  editFlowRef.value = false

  editAppStore.value = null
}

async function saveEditButton() {
  if (spinnerSaveEditRef.value) return

  const preExistingData = Object.keys(dataToPassRef.value).reduce(
    (acc, key) => {
      if (editableFields.includes(key)) {
        acc[key] = dataToPassRef.value[key]
      }
      return acc
    },
    {},
  )

  const normalizeData = (data, referenceData) => {
    const normalizedData = {}
    for (const key in data) {
      if (data[key] === '' || data[key] === undefined || data[key] === null) {
        // Treat empty strings, undefined, or null as non-existent keys
        if (!(key in referenceData)) {
          // Only add if the key doesn't exist in the reference data
          continue
        }
      }
      normalizedData[key] = data[key]
    }
    return normalizedData
  }

  const normalizedEditAppStore = normalizeData(
    editAppStore.value,
    preExistingData,
  )
  const normalizedPreExistingData = normalizeData(
    preExistingData,
    editAppStore.value,
  )

  if (
    JSON.stringify(normalizedEditAppStore) ===
    JSON.stringify(normalizedPreExistingData)
  ) {
    editAppStore.value = null
    editFlowRef.value = false
    return
  } else {
    spinnerSaveEditRef.value = true
    try {
      const numericFinancialData = filterNumericValues(editAppStore.value)

      if (!numericFinancialData) {
        console.error('Invalid financial data')
        spinnerSaveEditRef.value = false
        return
      }

      const res = await fetch(`${config.baseURL}/api/app/edit`, {
        method: 'PATCH',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          'X-CSRF-Token': getCSRFToken(),
        },
        body: JSON.stringify({
          ticker: ticker,
          period: periodRef.value,
          new_financial_data: numericFinancialData,
        }),
      })

      if (!res.ok) throw new Error('Failed to fetch')

      //   editAppStore.value = formatForRatios(editAppStore.value)
      //   dataToPassRef.value = editAppStore.value
      //   const newData = calculateRatios(editAppStore.value)
      //   tickerIsMounted.value.financial_data[periodRef.value] = newData

      //   editAppStore.value = null
      //   editFlowRef.value = false

      window.location.reload()
    } catch (error) {
      console.error(error)
    } finally {
      spinnerSaveEditRef.value = false
    }
  }
}

function toggleDeleteDropdown() {
  if (navSpinnerRef.value || spinnerDeleteRef.value) return
  isDeleteDropdownOpenRef.value = !isDeleteDropdownOpenRef.value
}

async function sendDeleteRequest() {
  if (
    navSpinnerRef.value ||
    spinnerDeleteRef.value ||
    errorAppRef.value ||
    spinnerAnalystRef.value ||
    !isDeleteDropdownOpenRef.value ||
    !ticker ||
    !periodRef.value ||
    !tickerIsMounted.value
  ) {
    return
  }

  try {
    spinnerDeleteRef.value = true
    const res = await fetch(
      `${config.baseURL}/api/app/delete-ticker?ticker=${ticker}&period=${periodRef.value}`,
      {
        method: 'DELETE',
        credentials: 'include',
        headers: {
          'X-CSRF-Token': getCSRFToken(),
        },
      },
    )

    if (!res.ok) throw new Error('Failed to fetch')

    let financialDataArray = Object.keys(tickerIsMounted.value.financial_data)
    const currentIndex = financialDataArray.indexOf(periodRef.value)

    // If this is the last period, clean up and redirect
    if (financialDataArray.length === 1 && cursorTickerStore.value === null) {
      tickerIsMounted.value = null
      if (appIsMountedStore.value) {
        appIsMountedStore.value = appIsMountedStore.value.filter(
          item => item !== ticker,
        )
        if (appIsMountedStore.value.length === 0) {
          appIsMountedStore.value = null
        }
      }
      tickerStore.value = ''
      tickerSeachResultStore.value = null
      router.push('/app')
    } else {
      const periodToDelete = periodRef.value
      if (currentIndex > 0) {
        await navPage('next')
      } else {
        await navPage('prev')
      }
      delete tickerIsMounted.value.financial_data[periodToDelete]
      financialDataArray = Object.keys(tickerIsMounted.value.financial_data)
    }
  } catch (error) {
    console.error('Failed to delete period:', error)
  } finally {
    spinnerDeleteRef.value = false
    isDeleteDropdownOpenRef.value = false
  }
}

function formatScore(score) {
  if (score === 0) {
    return 0.0
  }
  if (!score || isNaN(score)) {
    return 'score'
  } else {
    return score.toLocaleString('en-US', {
      minimumFractionDigits: 2,
      maximumFractionDigits: 2,
    })
  }
}

async function navPage(direction) {
  if (
    navSpinnerRef.value ||
    spinnerAnalystRef.value ||
    errorAppRef.value ||
    !ticker ||
    !tickerIsMounted.value
  ) {
    return
  }

  const financialDataArray = Object.keys(tickerIsMounted.value.financial_data)
  const index = financialDataArray.indexOf(periodRef.value)

  if (direction === 'prev') {
    if (index === financialDataArray.length - 1) {
      if (cursorTickerStore.value) {
        navSpinnerRef.value = true
        await fetchTickerData()
        navSpinnerRef.value = false
      }
    } else {
      periodRef.value = financialDataArray[index + 1]
    }
  }

  if (direction === 'next') {
    if (index > 0) {
      periodRef.value = financialDataArray[index - 1]
    }
  }
}

function displayErrorApp(message, timeout = 500) {
  errorMsgRef.value = message || 'An error has occurred'
  errorAppRef.value = true
  setTimeout(() => {
    errorAppRef.value = false
    errorMsgRef.value = ''
  }, timeout)
}

async function sendAnalystRequest() {
  if (
    navSpinnerRef.value ||
    spinnerDeleteRef.value ||
    spinnerAnalystRef.value ||
    errorAppRef.value ||
    isDeleteDropdownOpenRef.value ||
    !ticker ||
    !tickerIsMounted.value ||
    !tickerIsMounted.value.currency
  ) {
    return
  }

  if (!validateTicker(ticker)) {
    displayErrorApp('Invalid ticker')
    return
  }

  if (!validateCurrency(tickerIsMounted.value.currency)) {
    displayErrorApp('Invalid currency')
    return
  }

  spinnerAnalystRef.value = true
  try {
    const res = await fetch(`${config.baseURL}/api/app/analyst`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        Accept: 'application/json',
        'Content-Type': 'application/json',
        'X-CSRF-Token': getCSRFToken(),
      },
      body: JSON.stringify({
        ticker: ticker,
        currency: tickerIsMounted.value.currency,
      }),
    })

    if (!res.ok) {
      let errorMessage
      switch (res.status) {
        case 403:
          errorMessage = await res.text()
          break
        case 429: {
          errorMessage =
            'You are making too many requests. Please wait a few minutes.'
          break
        }
        default:
          errorMessage = ''
      }
      throw new Error(errorMessage)
    }

    const data = await res.json()
    tickerIsMounted.value.analyst = data.analyst_message
  } catch (error) {
    displayErrorApp(error.message, 5000)
  } finally {
    spinnerAnalystRef.value = false
  }
}

const calculatedRevenue = computed(() =>
  revenueNeededForWishedPER(
    customPriceRef.value,
    customPERRef.value,
    dataToPassRef.value.shares,
    tickerIsMounted.value.currency,
    dataToPassRef.value.net_income,
  ),
)

useHead({
  title: `nodo.finance - App: ${ticker}`,
  meta: [
    {
      name: 'twitter:title',
      content: `nodo.finance - App: ${ticker}`,
    },
    {
      property: 'og:title',
      content: `nodo.finance - App: ${ticker}`,
    },
    {
      name: 'robots',
      content: 'index, follow',
    },
  ],
})
</script>

<template>
  <div v-if="!tickerIsMounted" class="global-spinner">
    <LoadingWithinBox />
  </div>
  <div v-else>
    <div class="main-header">
      <h1 class="template-title" v-if="windowWidthRef > 1050 || editFlowRef">
        {{ ticker }}
      </h1>
      <div class="mobile-header" v-else>
        <h1 class="template-title">{{ ticker }}: {{ periodRef }}</h1>

        <div class="horizontal-input-mobile">
          <input
            :placeholder="
              tickerIsMounted.currency !== 'NA'
                ? tickerIsMounted.currency
                : '...'
            "
            class="price-input"
            type="number"
            name="price"
            v-model.number="customPriceRef"
            min="0"
            autocomplete="off"
            @keydown="e => e.key === '-' && e.preventDefault()"
          />
          <span>→</span>
          <span
            :class="{
              score: true,
              neutral:
                (!dataToPassRef.score || isNaN(dataToPassRef.score)) &&
                dataToPassRef.score != 0,
            }"
          >
            {{ formatScore(dataToPassRef.score) }}
          </span>
        </div>
        <div class="horizontal-input-mobile">
          <input
            placeholder="P/E"
            class="price-input"
            type="number"
            name="price"
            v-model.number="customPERRef"
            min="0"
            autocomplete="off"
            @keydown="e => e.key === '-' && e.preventDefault()"
          />
          <span>→</span>
          <span
            class="per"
            :class="{
              score: true,
              neutral: !customPERRef || dataToPassRef.eps <= 0,
            }"
          >
            {{
              wishedPER(
                customPERRef,
                dataToPassRef.eps,
                tickerIsMounted.currency,
              )
            }}
          </span>
        </div>
        <span
          class="revenue-mobile"
          :class="{
            score: true,
            neutral: !customPERRef || !customPriceRef,
          }"
        >
          {{ calculatedRevenue.value }}
          <sub
            v-show="calculatedRevenue.percentChange !== null"
            class="income-percentage"
            :class="{
              positive: calculatedRevenue.percentChange > 0,
              negative: calculatedRevenue.percentChange < 0,
            }"
          >
            {{ calculatedRevenue.percentChange?.toFixed(1) }}%
          </sub>
        </span>
      </div>
    </div>
    <div class="content-wrapper">
      <LoadingWithinBox v-if="!dataToPassRef" class="spinner" />
      <div v-else class="element1">
        <div v-if="!editFlowRef" class="element1">
          <div class="horizontal">
            <div class="navigation">
              <button
                aria-label="Previous"
                @click="navPage('prev')"
                :disabled="
                  navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef
                "
                :class="{
                  'disabled-overlay':
                    navSpinnerRef ||
                    spinnerDeleteRef ||
                    spinnerAnalystRef ||
                    spinnerSaveEditRef,
                }"
              >
                <SpinnerWhite v-if="navSpinnerRef" />
                <ArrowLeft v-else />
              </button>
              <span
                v-if="periodRef"
                v-show="windowWidthRef > 1050"
                class="period-span"
                >{{ periodRef }}</span
              >
              <SpinnerWhite v-else class="period-span" />
              <button
                aria-label="Next"
                @click="navPage('next')"
                :disabled="
                  navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef
                "
                :class="{
                  'disabled-overlay':
                    navSpinnerRef ||
                    spinnerDeleteRef ||
                    spinnerAnalystRef ||
                    spinnerSaveEditRef,
                }"
              >
                <ArrowRight />
              </button>
            </div>
            <button
              aria-label="Edit"
              class="finances"
              @click.prevent.stop="editButton"
              :disabled="navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef"
              :class="{
                'disabled-overlay':
                  navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef,
              }"
            >
              <EditIcon />
            </button>

            <div class="select-wrapper">
              <button
                aria-label="Delete"
                class="finances"
                :class="{
                  'disabled-overlay':
                    navSpinnerRef ||
                    spinnerDeleteRef ||
                    spinnerAnalystRef ||
                    spinnerSaveEditRef,
                }"
                @click.prevent.stop="toggleDeleteDropdown"
                :disabled="
                  navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef
                "
              >
                <TrashIcon />
              </button>

              <div v-if="isDeleteDropdownOpenRef">
                <div
                  class="backdrop"
                  @click.prevent.stop="toggleDeleteDropdown"
                ></div>
                <div class="custom-dropdown edit-buttons">
                  <button
                    aria-label="Cancel"
                    class="finances"
                    @click.prevent.stop="toggleDeleteDropdown"
                    :disabled="
                      navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef
                    "
                  >
                    <CloseWhite />
                  </button>
                  <button
                    aria-label="Delete"
                    class="finances"
                    @click.prevent.stop="sendDeleteRequest"
                    :disabled="
                      navSpinnerRef || spinnerDeleteRef || spinnerAnalystRef
                    "
                  >
                    <SpinnerWhite v-if="spinnerDeleteRef" />
                    <OkIcon v-else />
                  </button>
                </div>
              </div>
            </div>

            <div class="horizontal-input" v-show="windowWidthRef > 1050">
              <input
                :placeholder="
                  tickerIsMounted.currency !== 'NA'
                    ? tickerIsMounted.currency
                    : '...'
                "
                class="price-input"
                type="number"
                name="price"
                v-model.number="customPriceRef"
                min="0"
                autocomplete="off"
                @keydown="e => e.key === '-' && e.preventDefault()"
              />
              <span>→</span>
              <span
                :class="{
                  score: true,
                  neutral:
                    (!dataToPassRef.score || isNaN(dataToPassRef.score)) &&
                    dataToPassRef.score != 0,
                }"
              >
                {{ formatScore(dataToPassRef.score) }}
              </span>
            </div>
            <div class="horizontal-input" v-show="windowWidthRef > 1050">
              <input
                placeholder="P/E"
                class="price-input"
                type="number"
                name="price"
                v-model.number="customPERRef"
                min="0"
                autocomplete="off"
                @keydown="e => e.key === '-' && e.preventDefault()"
              />
              <span>→</span>
              <span
                class="per"
                :class="{
                  score: true,
                  neutral: !customPERRef || dataToPassRef.eps <= 0,
                }"
              >
                {{
                  wishedPER(
                    customPERRef,
                    dataToPassRef.eps,
                    tickerIsMounted.currency,
                  )
                }}
              </span>
              <span
                class="revenue"
                :class="{
                  score: true,
                  neutral: !customPERRef || !customPriceRef,
                }"
              >
                {{ calculatedRevenue.value }}
                <sub
                  v-show="calculatedRevenue.percentChange !== null"
                  class="income-percentage"
                  :class="{
                    positive: calculatedRevenue.percentChange > 0,
                    negative: calculatedRevenue.percentChange < 0,
                  }"
                >
                  {{ calculatedRevenue.percentChange?.toFixed(1) }}%
                </sub>
              </span>
            </div>

            <div class="submit-button-container">
              <button
                aria-label="Talk to analyst"
                class="talk-to-analyst finances"
                @click.prevent.stop="sendAnalystRequest"
                :disabled="
                  spinnerAnalystRef ||
                  errorAppRef ||
                  navSpinnerRef ||
                  spinnerDeleteRef
                "
                :class="{
                  'disabled-overlay':
                    navSpinnerRef || spinnerDeleteRef || spinnerSaveEditRef,
                }"
              >
                <CloseWhite v-if="errorAppRef" />
                <FaviconWhiteSpinner v-else-if="spinnerAnalystRef" />
                <FaviconWhite v-else class="icon" />
              </button>
              <p v-if="errorMsgRef" class="error-msg">{{ errorMsgRef }}</p>
            </div>
          </div>
          <div class="finances-wrapper">
            <AppRatios
              :dataToUse="dataToPassRef"
              :currency="tickerIsMounted.currency"
            />
            <div class="finances-column">
              <AppAnalyst
                :analyst="tickerIsMounted.analyst"
                :spinner="spinnerAnalystRef"
              />
              <AppSankey :dataToUse="dataToPassRef" />
            </div>
          </div>
        </div>
        <div v-else class="edit-wrapper">
          <div class="horizontal">
            <button
              aria-label="Cancel edit"
              class="finances"
              @click.prevent.stop="cancelEditButton"
              :disabled="spinnerSaveEditRef"
            >
              <CloseWhite />
            </button>
            <button
              aria-label="Save edit"
              class="finances"
              @click.prevent.stop="saveEditButton"
              :disabled="spinnerSaveEditRef"
            >
              <SpinnerWhite v-if="spinnerSaveEditRef" />
              <OkIcon v-else />
            </button>
          </div>
          <AppEdit />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.finances:disabled {
  position: relative;
  cursor: default;
}

.disabled-overlay {
  position: relative;
  cursor: default;
}

.disabled-overlay::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(128, 128, 128, 0.5);
  pointer-events: none;
  border-radius: 10px;
  cursor: default;
}

.submit-button-container {
  position: relative;
  height: 100%;
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
  z-index: 1000;
  text-align: left;
}

.revenue sub,
.revenue-mobile sub {
  vertical-align: sub;
  font-size: 10px;
  line-height: 0;
  position: relative;
  bottom: -5px;
  color: var(--gray-four);
  margin-left: 3px;
}

.income-percentage.positive {
  color: var(--green-one);
}
.income-percentage.negative {
  color: var(--red-one);
}

.horizontal {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  margin: 10px auto 15px;
  gap: 20px;
  background-color: var(--gray-two);
  border-radius: 10px;
  box-shadow: 0 0 10px var(--gray-two);
  width: fit-content;
}

.horizontal-input-mobile {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  margin: 0 auto;
  gap: 5px;
  background-color: var(--gray-two);
  border-radius: 10px;
  box-shadow: 0 0 10px var(--gray-two);
  width: fit-content;
}

.horizontal-input {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 5px;
  height: 100%;
}

.edit-buttons {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  margin: 0;
  gap: 10px;
}
.header {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 0;
  margin: 0 auto;
  gap: 20px;
}
.template-title {
  display: flex;
  margin: 0 auto;
  justify-content: center;
  align-items: center;
  background-color: var(--gray-two);
  border-radius: 10px;
  box-shadow: 0 0 10px var(--gray-two);
  width: fit-content;
  padding: 10px;
}
h1 {
  font-weight: 400;
  font-size: 20px;
  letter-spacing: 1px;
  margin: 0;
}
.content-wrapper {
  display: flex;
  justify-content: center;
  margin: 0 auto;
  gap: 50px;
}
.finances-wrapper {
  display: flex;
  justify-content: center;
  width: 100%;
  gap: 20px;
  padding-bottom: 80px;
}
.finances {
  display: flex;
  margin: 0;
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
  height: 100%;
  width: 40px;
  font-size: 16px;
}

.finances-column {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 20px;
  box-sizing: border-box;
  width: 400px;
  height: 100%;
}

.navigation {
  display: flex;
  gap: 5px;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.navigation button {
  border: none;
  border-radius: 10px;
  padding: 8px;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 40px;
  height: 100%;
  background-color: black;
  box-sizing: border-box;
}

.score {
  border: none;
  border-radius: 10px;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 55px;
  box-sizing: border-box;
  height: 100%;
  background-color: var(--gray-three);
  letter-spacing: 1.2px;
  color: black;
  font-weight: 550;
}

.per {
  width: 100px;
}
.revenue {
  width: 150px;
}
.revenue-mobile {
  height: 40px;
  width: 205px;
  margin-top: 10px;
}

.period-span {
  width: 65px;
  font-size: 14px;
  text-align: center;
  letter-spacing: 1px;
}
.spinner {
  display: flex;
  margin: 0 auto;
  align-items: center;
  height: 200px;
}
.global-spinner {
  display: flex;
  justify-content: center;
  align-items: center;
  margin: 50px auto 90px auto;
}
input {
  border: none;
  border-radius: 10px;
  width: 80px;
  height: 100%;
  padding: 0 5px;
  box-sizing: border-box;
  margin: 0;
  text-align: center;
  letter-spacing: 0.6px;
}
.negative {
  color: var(--red-one);
  font-size: 14px;
}

.neutral {
  color: var(--gray-four);
  font-size: 14px;
}

.positive {
  color: var(--green-one);
  font-size: 14px;
}

.inner-span {
  display: flex;
  align-items: center;
  gap: 10px;
}

.select-wrapper {
  position: relative;
  height: 100%;
}
.custom-dropdown {
  display: flex;
  position: absolute;
  background: white;
  border: none;
  border-radius: 10px;
  box-sizing: border-box;
  top: 100%;
  left: 0;
  padding: 10px;
  z-index: 10;
  height: 60px;
  box-shadow: 0 5px 10px rgba(0, 0, 0, 0.3);
}
.backdrop {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.3);
  z-index: 5;
}

.edit-wrapper {
  box-sizing: border-box;
}

.talk-to-analyst {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 100%;
  padding: 7px;
}
.element1 {
  width: 100%;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
}
.mobile-header {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.mobile-header .horizontal {
  padding: 0;
  margin: 0;
}
.mobile-header .template-title {
  padding: 10px;
  margin: 0;
  margin-bottom: 10px;
}

@media (max-width: 1050px) {
  .finances-column {
    max-width: 320px;
    width: 100%;
  }
}

@media (max-width: 700px) {
  .edit-wrapper .horizontal {
    margin: 5px auto;
  }
  .finances-wrapper {
    flex-direction: column;
    margin: 0 auto;
    justify-content: center;
    align-items: center;
  }

  .finances-column {
    flex-direction: column-reverse;
  }
  .content-wrapper {
    width: 90%;
  }
  .element1 {
    width: 100%;
  }
  .horizontal {
    padding: 10px 0;
    gap: 14px;
    margin-bottom: 12px;
    justify-content: space-around;
  }

  .template-title {
    margin: 0 auto;
  }
}

@media (max-width: 370px) {
  .error-msg {
    left: 20%;
    width: 100px;
  }
}
</style>
