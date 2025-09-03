import {
  WINDOW_SIZE,
  OVERLAP_STRIDE,
  BUFFER_SIZE,
  OUTPUT_CHUNK_SIZE,
  balance_indicators_en,
  income_indicators_en,
  cash_flow_indicators_en,
  balance_indicators_es,
  income_indicators_es,
  cash_flow_indicators_es,
} from './preprocessor_consts.js'
import { cleanChunk } from './cleaner.js'

export class Preprocessor {
  constructor() {
    this.WINDOW_SIZE = WINDOW_SIZE
    this.OVERLAP_STRIDE = OVERLAP_STRIDE
    this.BUFFER_SIZE = BUFFER_SIZE
    this.OUTPUT_CHUNK_SIZE = OUTPUT_CHUNK_SIZE

    this.balance_indicators_en_array = Array.from(balance_indicators_en)
    this.income_indicators_en_array = Array.from(income_indicators_en)
    this.cash_flow_indicators_en_array = Array.from(cash_flow_indicators_en)
    this.balance_indicators_es_array = Array.from(balance_indicators_es)
    this.income_indicators_es_array = Array.from(income_indicators_es)
    this.cash_flow_indicators_es_array = Array.from(cash_flow_indicators_es)
  }

  preprocess(content, period) {
    let balance_result = {}
    let income_result = {}
    let cash_flow_result = {}
    let language = 'EN'

    try {
      const lower_content = this.normalizeText(content)

      // try EN
      balance_result = this.findChunk(
        content,
        lower_content,
        this.balance_indicators_en_array,
      )
      income_result = this.findChunk(
        content,
        lower_content,
        this.income_indicators_en_array,
      )
      cash_flow_result = this.findChunk(
        content,
        lower_content,
        this.cash_flow_indicators_en_array,
      )

      // not enough hits -> try ES
      if (
        balance_result.metrics.first_unique_hits < 15 ||
        income_result.metrics.first_unique_hits < 15 ||
        cash_flow_result.metrics.first_unique_hits < 15
      ) {
        language = 'ES'
        balance_result = this.findChunk(
          content,
          lower_content,
          this.balance_indicators_es_array,
        )
        income_result = this.findChunk(
          content,
          lower_content,
          this.income_indicators_es_array,
        )
        cash_flow_result = this.findChunk(
          content,
          lower_content,
          this.cash_flow_indicators_es_array,
        )
      }

      // clean
      const balanceCleanResult = cleanChunk(
        'balance',
        balance_result.chunk,
        language,
        period,
      )
      balance_result.cleaned = balanceCleanResult.text
      balance_result.units = balanceCleanResult.units
      const incomeCleanResult = cleanChunk(
        'income',
        income_result.chunk,
        language,
        period,
      )
      income_result.cleaned = incomeCleanResult.text
      income_result.units = incomeCleanResult.units
      const cashFlowCleanResult = cleanChunk(
        'cash_flow',
        cash_flow_result.chunk,
        language,
        period,
      )
      cash_flow_result.cleaned = cashFlowCleanResult.text
      cash_flow_result.units = cashFlowCleanResult.units

      return {
        balance_result,
        income_result,
        cash_flow_result,
        language,
      }
    } catch (error) {
      console.error('Error in Preprocessor.preprocess:', error)
      throw new Error('Preprocessing failed')
    }
  }

  normalizeText(content) {
    // handle accent normalization
    return content
      .toLowerCase()
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
  }

  findChunk(content, lower_content, indicatorArray) {
    const OVERLAP_STRIDE = this.OVERLAP_STRIDE
    const WINDOW_SIZE = this.WINDOW_SIZE
    const BUFFER_SIZE = this.BUFFER_SIZE
    const OUTPUT_CHUNK_SIZE = this.OUTPUT_CHUNK_SIZE

    // Use typed arrays for better numeric performance
    const counts = new Uint32Array(5) // [first, second, third, fourth, fifth]
    let best_start = 0
    let best_indicators = new Set()

    const contentLength = lower_content.length

    const found_indicators = new Set()

    for (let i = 0; i < contentLength; i += OVERLAP_STRIDE) {
      found_indicators.clear()

      const windowEnd = Math.min(i + WINDOW_SIZE, contentLength)
      const window = lower_content.substring(i, windowEnd)

      // Check each indicator against the current window
      for (const indicator of indicatorArray) {
        if (window.includes(indicator)) {
          found_indicators.add(indicator)
        }
      }

      const count = found_indicators.size

      // Update top 5 counts
      if (count > counts[0]) {
        counts[4] = counts[3]
        counts[3] = counts[2]
        counts[2] = counts[1]
        counts[1] = counts[0]
        counts[0] = count
        best_start = i
        best_indicators = new Set(found_indicators)
      } else if (count > counts[1]) {
        counts[4] = counts[3]
        counts[3] = counts[2]
        counts[2] = counts[1]
        counts[1] = count
      } else if (count > counts[2]) {
        counts[4] = counts[3]
        counts[3] = counts[2]
        counts[2] = count
      } else if (count > counts[3]) {
        counts[4] = counts[3]
        counts[3] = count
      } else if (count > counts[4]) {
        counts[4] = count
      }
    }

    // Calculate chunk boundaries
    const chunk_start = best_start > BUFFER_SIZE ? best_start - BUFFER_SIZE : 0
    const chunk_length = Math.min(
      OUTPUT_CHUNK_SIZE,
      content.length - chunk_start,
    )
    const chunk = content.substring(chunk_start, chunk_start + chunk_length)

    return {
      chunk,
      metrics: {
        first_unique_hits: counts[0],
        second_unique_hits: counts[1],
        third_unique_hits: counts[2],
        fourth_unique_hits: counts[3],
        fifth_unique_hits: counts[4],
      },
      indicators: Array.from(best_indicators),
    }
  }
}
