import { reactive } from 'vue'

export const editAppStore = reactive({
  value: '',
})
export const appIsMountedStore = reactive({
  value: null,
})
export const tickerSeachResultStore = reactive({
  value: null,
})
export const tickerStore = reactive({
  value: '',
})
export const cursorTickerStore = reactive({
  value: null,
})
export const urlStore = reactive({
  value: '',
})
export const periodStore = reactive({
  value: '',
})
export const spinnerAppStore = reactive({
  value: false,
})
export const parserWorkingStore = reactive({
  value: null,
})
export const currencyStore = reactive({
  value: 'USD',
})
export const submitTypeStore = reactive({
  value: 'file',
})
export const fileNameStore = reactive({
  value: '',
})
