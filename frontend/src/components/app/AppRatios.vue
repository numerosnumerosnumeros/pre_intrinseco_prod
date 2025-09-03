<script setup>
const { dataToUse, currency } = defineProps({
  dataToUse: {
    type: Object,
    required: true,
  },
  currency: {
    type: String,
    required: true,
  },
})

// format color
function valueClass(key) {
  return formatValue(key) === 'NA' ? 'na-value' : ''
}
// format number
function formatValue(key) {
  const value = dataToUse[key]

  if (!value || isNaN(value)) {
    return 'NA'
  }

  const currencyDisplay =
    currency === 'USD'
      ? '$'
      : currency === 'EUR'
        ? '€'
        : currency === 'GBP'
          ? '£'
          : ''

  switch (key) {
    case 'enterpriseValue':
    case 'current_assets':
    case 'cash_and_equivalents':
    case 'non_current_assets':
    case 'total_assets':
    case 'current_liabilities':
    case 'non_current_liabilities':
    case 'total_liabilities':
    case 'equity':
    case 'working_capital':
    case 'net_debt':
    case 'revenue':
    case 'net_income':
    case 'cash_flow_from_operations':
    case 'cash_flow_from_investing':
    case 'cash_flow_from_financing':
      return `${value.toLocaleString('en-US', {
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
      })} ${currencyDisplay} `
    case 'book_value':
    case 'eps':
      return `${value.toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })} ${currencyDisplay}`
    case 'net_margin':
    case 'roa':
    case 'roe':
      return `${(value * 100).toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })} %`
    case 'shares':
      return value.toLocaleString('en-US', {
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
      })
    case 'peRatio':
    case 'pbRatio':
    case 'evOcf':
    case 'evNetIncome':
    case 'evMarketCap':
    case 'wc_ncl':
    case 'liquidity':
    case 'leverage':
    case 'solvency':
      return value.toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })
    default:
      return value.toLocaleString('en-US', {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      })
  }
}

const calculateChange = key => {
  const current_value = dataToUse[key]
  const prev_value = dataToUse[key + '_prev']

  if (
    !prev_value ||
    isNaN(prev_value) ||
    prev_value == 0 ||
    !current_value ||
    isNaN(current_value) ||
    current_value == 0
  ) {
    return ''
  }

  if (prev_value < 0 && current_value < 0) {
    return (
      Math.round(
        ((Math.abs(prev_value) - Math.abs(current_value)) /
          Math.abs(prev_value)) *
          100 *
          10,
      ) / 10
    )
  }

  if (prev_value < 0 && current_value > 0) {
    return Math.abs(
      Math.round(((current_value - prev_value) / prev_value) * 100 * 10) / 10,
    )
  }

  return Math.round(((current_value - prev_value) / prev_value) * 100 * 10) / 10
}

const getChangeColorClass = key => {
  const greyKeys = [
    'enterpriseValue',
    'evOcf',
    'peRatio',
    'pbRatio',
    'evMarketCap',
    'evNetIncome',
  ]
  if (greyKeys.includes(key)) {
    return {
      'change-green': false,
      'change-gray': true,
      'change-red': false,
    }
  }
  const change = calculateChange(key)
  return {
    'change-green': change > 0,
    'change-gray': change === 0,
    'change-red': change < 0,
  }
}

const formatChange = key => {
  const change = calculateChange(key)
  if (change === '') return ''
  return `<sub>${change}%</sub>`
}
</script>

<template>
  <div class="wrapper" v-if="dataToUse">
    <div class="whole-width main-ratios">
      <div class="row">
        <div class="item">
          <div class="data-key">P / E</div>
          <div :class="valueClass('peRatio')" class="data-value">
            <span
              v-if="formatChange('peRatio')"
              :class="getChangeColorClass('peRatio')"
              v-html="formatChange('peRatio')"
            >
            </span>
            {{ formatValue('peRatio') }}
          </div>
        </div>

        <div class="item">
          <div class="data-key">P / BV</div>
          <div :class="valueClass('pbRatio')" class="data-value">
            <span
              v-if="formatChange('pbRatio')"
              :class="getChangeColorClass('pbRatio')"
              v-html="formatChange('pbRatio')"
            >
            </span>
            {{ formatValue('pbRatio') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">EV</div>
          <div :class="valueClass('enterpriseValue')" class="data-value">
            <span
              v-if="formatChange('enterpriseValue')"
              :class="getChangeColorClass('enterpriseValue')"
              v-html="formatChange('enterpriseValue')"
            >
            </span
            >{{ formatValue('enterpriseValue') }}
          </div>
        </div>

        <div class="item">
          <div class="data-key">EV<sub>cap</sub></div>
          <div :class="valueClass('evMarketCap')" class="data-value">
            <span
              v-if="formatChange('evMarketCap')"
              :class="getChangeColorClass('evMarketCap')"
              v-html="formatChange('evMarketCap')"
            >
            </span
            >{{ formatValue('evMarketCap') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">EV / CFO</div>
          <div :class="valueClass('evOcf')" class="data-value">
            <span
              v-if="formatChange('evOcf')"
              :class="getChangeColorClass('evOcf')"
              v-html="formatChange('evOcf')"
            >
            </span
            >{{ formatValue('evOcf') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">EV / π</div>
          <div :class="valueClass('evNetIncome')" class="data-value">
            <span
              v-if="formatChange('evNetIncome')"
              :class="getChangeColorClass('evNetIncome')"
              v-html="formatChange('evNetIncome')"
            >
            </span
            >{{ formatValue('evNetIncome') }}
          </div>
        </div>
      </div>
    </div>
    <div class="whole-width">
      <div class="row">
        <div class="item">
          <div class="data-key">CA</div>
          <div :class="valueClass('current_assets')" class="data-value">
            <span
              v-if="formatChange('current_assets')"
              :class="getChangeColorClass('current_assets')"
              v-html="formatChange('current_assets')"
            >
            </span
            >{{ formatValue('current_assets') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">NCA</div>
          <div :class="valueClass('non_current_assets')" class="data-value">
            <span
              v-if="formatChange('non_current_assets')"
              :class="getChangeColorClass('non_current_assets')"
              v-html="formatChange('non_current_assets')"
            >
            </span
            >{{ formatValue('non_current_assets') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">TA</div>
          <div :class="valueClass('total_assets')" class="data-value">
            <span
              v-if="formatChange('total_assets')"
              :class="getChangeColorClass('total_assets')"
              v-html="formatChange('total_assets')"
            >
            </span
            >{{ formatValue('total_assets') }}
          </div>
        </div>

        <div class="item">
          <div class="data-key">Cash</div>
          <div :class="valueClass('cash_and_equivalents')" class="data-value">
            <span
              v-if="formatChange('cash_and_equivalents')"
              :class="getChangeColorClass('cash_and_equivalents')"
              v-html="formatChange('cash_and_equivalents')"
            >
            </span
            >{{ formatValue('cash_and_equivalents') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">CL</div>
          <div :class="valueClass('current_liabilities')" class="data-value">
            <span
              v-if="formatChange('current_liabilities')"
              :class="getChangeColorClass('current_liabilities')"
              v-html="formatChange('current_liabilities')"
            >
            </span
            >{{ formatValue('current_liabilities') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">NCL</div>
          <div
            :class="valueClass('non_current_liabilities')"
            class="data-value"
          >
            <span
              v-if="formatChange('non_current_liabilities')"
              :class="getChangeColorClass('non_current_liabilities')"
              v-html="formatChange('non_current_liabilities')"
            >
            </span
            >{{ formatValue('non_current_liabilities') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">TL</div>
          <div :class="valueClass('total_liabilities')" class="data-value">
            <span
              v-if="formatChange('total_liabilities')"
              :class="getChangeColorClass('total_liabilities')"
              v-html="formatChange('total_liabilities')"
            >
            </span
            >{{ formatValue('total_liabilities') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">E</div>
          <div :class="valueClass('equity')" class="data-value">
            <span
              v-if="formatChange('equity')"
              :class="getChangeColorClass('equity')"
              v-html="formatChange('equity')"
            >
            </span
            >{{ formatValue('equity') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">WC</div>
          <div :class="valueClass('working_capital')" class="data-value">
            <span
              v-if="formatChange('working_capital')"
              :class="getChangeColorClass('working_capital')"
              v-html="formatChange('working_capital')"
            >
            </span
            >{{ formatValue('working_capital') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">WC / NCL</div>
          <div :class="valueClass('wc_ncl')" class="data-value">
            <span
              v-if="formatChange('wc_ncl')"
              :class="getChangeColorClass('wc_ncl')"
              v-html="formatChange('wc_ncl')"
            >
            </span
            >{{ formatValue('wc_ncl') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">shares<sub>≈</sub></div>
          <div :class="valueClass('shares')" class="data-value">
            <span
              v-if="formatChange('shares')"
              :class="getChangeColorClass('shares')"
              v-html="formatChange('shares')"
            >
            </span
            >{{ formatValue('shares') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">BV</div>
          <div :class="valueClass('book_value')" class="data-value">
            <span
              v-if="formatChange('book_value')"
              :class="getChangeColorClass('book_value')"
              v-html="formatChange('book_value')"
            >
            </span
            >{{ formatValue('book_value') }}
          </div>
        </div>
      </div>
    </div>

    <div class="whole-width">
      <div class="row">
        <div class="item">
          <div class="data-key">R</div>
          <div :class="valueClass('revenue')" class="data-value">
            <span
              v-if="formatChange('revenue')"
              :class="getChangeColorClass('revenue')"
              v-html="formatChange('revenue')"
            >
            </span
            >{{ formatValue('revenue') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">π</div>
          <div :class="valueClass('net_income')" class="data-value">
            <span
              v-if="formatChange('net_income')"
              :class="getChangeColorClass('net_income')"
              v-html="formatChange('net_income')"
            >
            </span
            >{{ formatValue('net_income') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">EPS</div>
          <div :class="valueClass('eps')" class="data-value">
            <span
              v-if="formatChange('eps')"
              :class="getChangeColorClass('eps')"
              v-html="formatChange('eps')"
            >
            </span
            >{{ formatValue('eps') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">Margin<sub>net</sub></div>
          <div :class="valueClass('net_margin')" class="data-value">
            <span
              v-if="formatChange('net_margin')"
              :class="getChangeColorClass('net_margin')"
              v-html="formatChange('net_margin')"
            >
            </span
            >{{ formatValue('net_margin') }}
          </div>
        </div>
      </div>
      <div class="row">
        <div class="item">
          <div class="data-key">ROA</div>
          <div :class="valueClass('roa')" class="data-value">
            <span
              v-if="formatChange('roa')"
              :class="getChangeColorClass('roa')"
              v-html="formatChange('roa')"
            >
            </span
            >{{ formatValue('roa') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">ROE</div>
          <div :class="valueClass('roe')" class="data-value">
            <span
              v-if="formatChange('roe')"
              :class="getChangeColorClass('roe')"
              v-html="formatChange('roe')"
            >
            </span
            >{{ formatValue('roe') }}
          </div>
        </div>
      </div>
    </div>

    <div class="columns-wrapper">
      <div class="column">
        <div class="item">
          <div class="data-key">Liquidity</div>
          <div :class="valueClass('liquidity')" class="data-value">
            <span
              v-if="formatChange('liquidity')"
              :class="getChangeColorClass('liquidity')"
              v-html="formatChange('liquidity')"
            >
            </span
            >{{ formatValue('liquidity') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">Solvency</div>
          <div :class="valueClass('solvency')" class="data-value">
            <span
              v-if="formatChange('solvency')"
              :class="getChangeColorClass('solvency')"
              v-html="formatChange('solvency')"
            >
            </span
            >{{ formatValue('solvency') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">Leverage</div>
          <div :class="valueClass('leverage')" class="data-value">
            <span
              v-if="formatChange('leverage')"
              :class="getChangeColorClass('leverage')"
              v-html="formatChange('leverage')"
            >
            </span
            >{{ formatValue('leverage') }}
          </div>
        </div>
      </div>

      <div class="column">
        <div class="item">
          <div class="data-key">CF<sub>op</sub></div>
          <div
            :class="valueClass('cash_flow_from_operations')"
            class="data-value"
          >
            <span
              v-if="formatChange('cash_flow_from_operations')"
              :class="getChangeColorClass('cash_flow_from_operations')"
              v-html="formatChange('cash_flow_from_operations')"
            >
            </span
            >{{ formatValue('cash_flow_from_operations') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">CF<sub>inv</sub></div>
          <div
            :class="valueClass('cash_flow_from_investing')"
            class="data-value"
          >
            <span
              v-if="formatChange('cash_flow_from_investing')"
              :class="getChangeColorClass('cash_flow_from_investing')"
              v-html="formatChange('cash_flow_from_investing')"
            >
            </span
            >{{ formatValue('cash_flow_from_investing') }}
          </div>
        </div>
        <div class="item">
          <div class="data-key">CF<sub>fin</sub></div>
          <div
            :class="valueClass('cash_flow_from_financing')"
            class="data-value"
          >
            <span
              v-if="formatChange('cash_flow_from_financing')"
              :class="getChangeColorClass('cash_flow_from_financing')"
              v-html="formatChange('cash_flow_from_financing')"
            >
            </span
            >{{ formatValue('cash_flow_from_financing') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.change-green {
  font-size: 10px;
  color: var(--green-one);
  padding-right: 3px;
}
.change-gray {
  font-size: 10px;
  color: var(--gray-four);
  padding-right: 3px;
}
.change-red {
  font-size: 10px;
  color: var(--red-one);
  padding-right: 3px;
}

.wrapper {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
  align-items: center;
  width: 580px;
  margin: 0;
}

.whole-width {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 15px 10px;
  margin: 0;
  justify-content: center;
  border-radius: 15px;
  border: none;
  width: 100%;
  box-sizing: border-box;
  background-color: var(--gray-one);
}

.main-ratios {
  /* border: 3px solid black; */
  box-sizing: border-box;
}

.item:hover {
  background-color: var(--gray-two);
  border-radius: 8px;
  cursor: default;
}

.row {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  width: 100%;
}

.item {
  display: flex;
  justify-content: space-between;
  padding: 5px 10px;
  align-items: center;
  text-align: left;
  font-size: 16px;
  height: 24px;
  width: 240px;
  letter-spacing: 0.6px;
}

.columns-wrapper {
  display: flex;
  justify-content: center;
  gap: 10px;
  width: 100%;
}

.column {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 15px 10px;
  margin: 0;
  justify-content: center;
  border-radius: 15px;
  border: none;
  width: 100%;
  box-sizing: border-box;
  background-color: var(--gray-one);
}

.data-key {
  font-weight: bold;
}
.na-value {
  color: var(--gray-five);
}

sub {
  font-size: 8px;
}

@media (max-width: 1050px) {
  .wrapper {
    width: 100%;
    max-width: 320px;
  }
  .whole-width {
    width: 100%;
  }
  .item {
    width: 100%;
    box-sizing: border-box;
    font-size: 16px;
  }
  .row {
    flex-direction: column;
    gap: 10px;
    width: 100%;
  }
  .columns-wrapper {
    flex-direction: column;
    gap: 10px;
    width: 100%;
  }
  .column {
    width: 100%;
  }

  @media (max-width: 310px) {
    .item {
      font-size: 15px;
    }
  }
}
</style>
