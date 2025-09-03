<script setup>
import { editAppStore } from '@state/app.js'
import { watch } from 'vue'

const SAFE_MAX = 1e14 // 100 trillion
const SAFE_MIN = -1e14 // -100 trillion
const EPS_MAX = 1e5 // $100,000 per share
const EPS_MIN = -1e5 // -$100,000 per share

const fieldConstraints = {
  current_assets: {
    decimals: 0,
    minValue: 0,
    maxValue: SAFE_MAX,
    allowNegative: false,
  },
  non_current_assets: {
    decimals: 0,
    minValue: 0,
    maxValue: SAFE_MAX,
    allowNegative: false,
  },
  eps: {
    decimals: 4,
    minValue: EPS_MIN,
    maxValue: EPS_MAX,
    allowNegative: true,
  },
  cash_and_equivalents: {
    decimals: 0,
    minValue: 0,
    maxValue: SAFE_MAX,
    allowNegative: false,
  },
  cash_flow_from_financing: {
    decimals: 0,
    minValue: SAFE_MIN,
    maxValue: SAFE_MAX,
    allowNegative: true,
  },
  cash_flow_from_investing: {
    decimals: 0,
    minValue: SAFE_MIN,
    maxValue: SAFE_MAX,
    allowNegative: true,
  },
  cash_flow_from_operations: {
    decimals: 0,
    minValue: SAFE_MIN,
    maxValue: SAFE_MAX,
    allowNegative: true,
  },
  revenue: {
    decimals: 0,
    minValue: 0,
    maxValue: SAFE_MAX,
    allowNegative: false,
  },
  current_liabilities: {
    decimals: 0,
    minValue: 0,
    maxValue: SAFE_MAX,
    allowNegative: false,
  },
  non_current_liabilities: {
    decimals: 0,
    minValue: 0,
    maxValue: SAFE_MAX,
    allowNegative: false,
  },
  net_income: {
    decimals: 0,
    minValue: SAFE_MIN,
    maxValue: SAFE_MAX,
    allowNegative: true,
  },
}

const formatting = {
  current_assets: 'CA',
  non_current_assets: 'NCA',
  cash_and_equivalents: 'Cash',
  current_liabilities: 'CL',
  non_current_liabilities: 'NCL',
  revenue: 'R',
  net_income: 'π',
  eps: 'EPS',
  cash_flow_from_operations: 'Op. CF',
  cash_flow_from_investing: 'Inv. CF',
  cash_flow_from_financing: 'Fin. CF',
}

const sanitizeValue = (value, field) => {
  const constraints = fieldConstraints[field]

  if (value === '-' && constraints.allowNegative) return '-'

  if (!value && value !== 0) return ''

  value = String(value)
  value = value.replace(/\s/g, '') // whitespace

  if (field === 'eps') {
    value = value.replace(/,/g, '.') // commas -> dots
    const [integer, ...decimals] = value.split('.')
    // Trim leading zeros from integer part, but keep single zero
    const trimmedInteger = integer.replace(/^0+(\d)/, '$1')
    value = decimals.length
      ? `${trimmedInteger}.${decimals.join('')}`
      : trimmedInteger
    // Remove non-numeric characters (keeping dots)
    value = value.replace(/[^0-9.-]/g, '')
  } else {
    value = value.replace(/[^0-9-]/g, '')
    // Trim leading zeros for non-eps fields, but keep single zero
    if (value.startsWith('0') && value.length > 1 && !value.startsWith('0.')) {
      value = value.replace(/^0+/, '')
    }
  }

  // Handle minus signs based on field type
  if (!constraints.allowNegative) {
    value = value.replace(/-/g, '') // Remove all minus signs for non-negative fields
  } else if (value.startsWith('-')) {
    value = '-' + value.slice(1).replace(/-/g, '') // Keep only first minus sign
    // Trim leading zeros after minus sign
    if (
      value.startsWith('-0') &&
      value.length > 2 &&
      !value.startsWith('-0.')
    ) {
      value = '-' + value.slice(2).replace(/^0+/, '')
    }
  } else {
    value = value.replace(/-/g, '')
  }

  // Validate number
  const numValue = Number(value)
  if (value && !Number.isFinite(numValue)) return ''

  // Check range
  if (numValue > constraints.maxValue || numValue < constraints.minValue) {
    return value.slice(0, -1) // Just remove the last character
  }

  // Check decimals
  if (constraints.decimals === 0) {
    if (!Number.isInteger(numValue)) return ''
  } else {
    const [integerPart, decimalPart = ''] = value.split('.')
    if (decimalPart.length > constraints.decimals) {
      value = integerPart + '.' + decimalPart.slice(0, constraints.decimals)
    }
  }

  return value
}

// Watch each property individually
Object.keys(formatting).forEach(key => {
  watch(
    () => editAppStore.value[key],
    newValue => {
      if (newValue === null || newValue === undefined) return
      const sanitized = sanitizeValue(newValue, key)
      if (sanitized !== newValue) {
        editAppStore.value[key] = sanitized
      }
    },
  )
})
</script>

<template>
  <div class="financial-data-container">
    <div v-for="(shortValue, key) in formatting" :key="key" class="data-entry">
      <div class="data-key">{{ shortValue }}</div>
      <div class="data-value">
        <input
          :value="editAppStore.value[key]"
          @input="e => (editAppStore.value[key] = e.target.value)"
          type="text"
          :name="key"
          class="data-input"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.financial-data-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  border: none;
  border-radius: 30px;
  padding: 25px;
  background-color: var(--gray-two);
  box-shadow: 0 0 10px 0 var(--gray-two);
  box-sizing: border-box;
}

.data-entry {
  display: flex;
  justify-content: space-between;
  padding: 8px 10px;
  gap: 5px;
  align-items: center;
  text-align: left;
  font-size: 14px;
}

.data-entry:hover,
.data-entry:focus-within {
  background-color: var(--gray-three);
  border-radius: 6px;
}

.data-input:hover,
.data-input:focus {
  cursor: text;
  outline: none;
}

.data-key {
  width: 105px;
  font-weight: bold;
}

.data-input {
  width: 110px;
  text-align: right;
  letter-spacing: 1px;
  padding: 0;
  margin: 0;
  font-size: 15px;
}

.data-input::-webkit-outer-spin-button,
.data-input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}
.data-input[type='number'] {
  -moz-appearance: textfield;
  appearance: textfield;
}
input {
  background-color: transparent;
  padding: 10px 15px;
}

@media (max-width: 680px) {
  .financial-data-container {
    padding: 10px 25px;
  }
}
@media (max-width: 570px) {
  .financial-data-container {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 325px) {
  .financial-data-container {
    border-radius: 0;
  }
}
</style>
