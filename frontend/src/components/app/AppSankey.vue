<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import * as d3 from 'd3'
import { sankey, sankeyLinkHorizontal } from 'd3-sankey'
const { dataToUse } = defineProps({
  dataToUse: {
    type: Object,
    required: true,
  },
})

const invalidDataRef = ref(false)

const nodeColorMap = {
  Cash: 'var(--green-one)',
  CA: 'var(--green-one)',
  NCA: 'var(--green-one)',
  CL: 'var(--pink-two)',
  NCL: 'var(--pink-two)',
  π: 'var(--green-one)',
  Costs: 'var(--pink-two)',
}

const createSankeyData = () => {
  const costsValue = Math.abs(dataToUse.revenue - dataToUse.net_income)

  const normalizeValue = value => {
    return Math.pow(value, 0.7) // Fractional power transformation (λ=0.7)
  }

  const baseNodes = [
    { name: 'Cash' },
    { name: 'CA' },
    { name: 'NCA' },
    { name: 'TA' },
    { name: 'CL' },
    { name: 'NCL' },
    { name: 'TL' },
    { name: 'R' },
    { name: 'π' },
    { name: 'Costs' },
  ]

  // Only remove π node if net_income is negative
  const nodes =
    dataToUse.net_income >= 0
      ? baseNodes
      : baseNodes.filter(node => node.name !== 'π')

  // Create base links array
  const baseLinks = [
    {
      source: 'Cash',
      target: 'CA',
      value: normalizeValue(dataToUse.cash_and_equivalents),
    },
    {
      source: 'CA',
      target: 'TA',
      value: normalizeValue(dataToUse.current_assets),
    },
    {
      source: 'NCA',
      target: 'TA',
      value: normalizeValue(dataToUse.non_current_assets),
    },
    {
      source: 'CL',
      target: 'TL',
      value: normalizeValue(dataToUse.current_liabilities),
    },
    {
      source: 'NCL',
      target: 'TL',
      value: normalizeValue(dataToUse.non_current_liabilities),
    },
    { source: 'Costs', target: 'R', value: normalizeValue(costsValue) },
  ]

  const links =
    dataToUse.net_income >= 0
      ? [
          ...baseLinks,
          {
            source: 'π',
            target: 'R',
            value: normalizeValue(dataToUse.net_income),
          },
        ]
      : baseLinks

  return {
    nodes,
    links,
  }
}

const validateSankeyData = data => {
  const requiredFields = [
    'cash_and_equivalents',
    'current_assets',
    'non_current_assets',
    'total_assets',
    'current_liabilities',
    'non_current_liabilities',
    'total_liabilities',
    'revenue',
    'net_income',
  ]

  // Check if all fields exist and are valid numbers
  const isValid = requiredFields.every(field => {
    return (
      field in data &&
      typeof data[field] === 'number' &&
      !isNaN(data[field]) &&
      isFinite(data[field])
    )
  })

  return isValid
}

const updateDimensions = () => {
  const currentWidth = window.innerWidth
  const width = currentWidth <= 360 ? 550 : currentWidth >= 1050 ? 600 : 700
  return width
}

const createSankeyDiagram = async () => {
  d3.select('#sankey').selectAll('*').remove()

  await nextTick()
  const wrapper = document.querySelector('.wrapper')
  if (!wrapper) return

  const width = updateDimensions()
  const height = wrapper.clientHeight

  // Get fresh data
  const sankeyData = createSankeyData()

  // Update SVG dimensions
  const svg = d3
    .select('#sankey')
    .append('svg')
    .attr('width', '100%')
    .attr('height', '100%')
    .attr('viewBox', `0 0 ${width} ${height}`)
    .append('g')
    .attr('transform', `translate(${0},${40})`)

  // Create Sankey generator with responsive values
  const sankeyGenerator = sankey()
    .nodeWidth('20')
    .nodePadding(50)
    .nodeId(d => d.name)
    .extent([
      [0, 0],
      [width - 15, height - 80],
    ])

  // Generate the sankey diagram
  const { nodes, links } = sankeyGenerator(sankeyData)

  const acNode = nodes.find(n => n.name === 'CA')
  if (acNode) {
    // How much to shift left
    const shiftAmount = width * 0.1 // 10% of chart width

    // Shift the AC node left
    const nodeWidth = acNode.x1 - acNode.x0
    acNode.x0 -= shiftAmount
    acNode.x1 = acNode.x0 + nodeWidth
  }

  // First, define gradients in your SVG defs
  const defs = svg.append('defs')

  // eslint-disable-next-line no-unused-vars
  nodes.forEach((node, i) => {
    const gradientId = `gradient-${node.name}`
    let baseColor = nodeColorMap[node.name] || '#808080'
    const endColor = baseColor.includes('red')
      ? 'var(--gray-two)'
      : 'var(--gray-two)'

    const gradient = defs
      .append('linearGradient')
      .attr('id', gradientId)
      .attr('gradientUnits', 'userSpaceOnUse')
      // Change to horizontal gradient for consistency
      .attr('x1', '0%')
      .attr('y1', '0%')
      .attr('x2', '100%')
      .attr('y2', '0%')

    gradient
      .append('stop')
      .attr('offset', '0%')
      .attr('stop-color', baseColor)
      .attr('stop-opacity', 1)

    gradient
      .append('stop')
      .attr('offset', '100%')
      .attr('stop-color', endColor)
      .attr('stop-opacity', 1)
  })

  // Modify stroke attribute to use gradients
  svg
    .append('g')
    .selectAll('path')
    .data(links)
    .join('path')
    .attr('d', sankeyLinkHorizontal())
    .attr('fill', 'none')
    .attr('stroke', d => {
      const sourceName = nodes[d.source.index].name
      return `url(#gradient-${sourceName})`
    })
    .attr('stroke-opacity', 0.6)
    .attr('stroke-width', d => Math.max(2, d.width))
    .style('transition', 'stroke-opacity 0.2s ease')

  svg
    .append('g')
    .selectAll('rect')
    .data(nodes)
    .join('rect')
    .attr('x', d => d.x0)
    .attr('y', d => d.y0)
    .attr('height', d => Math.max(2, d.y1 - d.y0))
    .attr('width', d => d.x1 - d.x0)
    .attr('fill', d => {
      if (d.x0 < width / 2) {
        return nodeColorMap[d.name]
      }
      return 'var(--gray-three)'
    })
    .attr('fill-opacity', 0.8)

  // Add labels with formatted values
  svg
    .append('g')
    .selectAll('text')
    .data(nodes)
    .join('text')
    .style('font-size', window.innerWidth < 1050 ? '32px' : '24px')
    .style('letter-spacing', '1.2px')
    .attr('x', d => {
      if (d.x0 >= width / 2) {
        return d.x1 + 5
      }
      return d.x1 + 5
    })
    .attr('y', d => (d.y1 + d.y0) / 2)
    .attr('dy', '0.35em')
    .attr('text-anchor', 'start')
    .text(d => {
      if (d.x0 >= width / 2) {
        return d.name
      }

      if (d.name === 'π' && dataToUse.net_income < 0) {
        return ''
      }

      let percentage
      if (d.name === 'Cash') {
        percentage = (
          (dataToUse.cash_and_equivalents / dataToUse.current_assets) *
          100
        ).toFixed(1)
      } else if (d.name === 'CA') {
        percentage = (
          (dataToUse.current_assets / dataToUse.total_assets) *
          100
        ).toFixed(1)
      } else if (d.name === 'NCA') {
        percentage = (
          (dataToUse.non_current_assets / dataToUse.total_assets) *
          100
        ).toFixed(1)
      } else if (d.name === 'CL') {
        percentage = (
          (dataToUse.current_liabilities / dataToUse.total_liabilities) *
          100
        ).toFixed(1)
      } else if (d.name === 'NCL') {
        percentage = (
          (dataToUse.non_current_liabilities / dataToUse.total_liabilities) *
          100
        ).toFixed(1)
      } else if (d.name === 'π') {
        percentage = ((dataToUse.net_income / dataToUse.revenue) * 100).toFixed(
          1,
        )
      } else if (d.name === 'Costs') {
        percentage = (
          (Math.abs(dataToUse.revenue - dataToUse.net_income) /
            dataToUse.revenue) *
          100
        ).toFixed(1)
      }

      return `${d.name} (${percentage}%)`
    })
}

onMounted(() => {
  if (!validateSankeyData(dataToUse)) {
    invalidDataRef.value = true
    return
  }

  let prevWidth = window.innerWidth
  let debounceTimeout

  const handleResize = () => {
    const currentWidth = window.innerWidth
    const crossedThreshold =
      (prevWidth <= 360 && currentWidth > 360) ||
      (prevWidth > 360 && currentWidth <= 360) ||
      (prevWidth < 1050 && currentWidth >= 1050) ||
      (prevWidth >= 1050 && currentWidth < 1050)

    if (crossedThreshold) {
      clearTimeout(debounceTimeout)
      debounceTimeout = setTimeout(() => {
        createSankeyDiagram()
        prevWidth = currentWidth
      }, 50)
    }
  }

  // Use both ResizeObserver and window resize event
  const resizeObserver = new ResizeObserver(handleResize)
  window.addEventListener('resize', handleResize)

  // Initial render
  createSankeyDiagram()

  // Observe after initial render
  nextTick(() => {
    const wrapper = document.querySelector('.wrapper')
    if (wrapper) {
      resizeObserver.observe(wrapper)
    }
  })

  // Cleanup
  onUnmounted(() => {
    resizeObserver.disconnect()
    window.removeEventListener('resize', handleResize)
    clearTimeout(debounceTimeout)
  })
})

watch(
  () => dataToUse,
  newData => {
    if (!validateSankeyData(newData)) {
      invalidDataRef.value = true
      return
    }
    createSankeyDiagram()
  },
  { deep: true },
)
</script>

<template>
  <div class="wrapper">
    <p v-if="invalidDataRef">Incomplete data</p>
    <div v-else id="sankey"></div>
  </div>
</template>

<style scoped>
.wrapper {
  padding: 0;
  margin: 0;
  background-color: var(--gray-one);
  height: 485px;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  border-radius: 15px;
  box-sizing: border-box;
}

#sankey {
  width: 100%;
  height: 100%;
}

p {
  letter-spacing: 0.6px;
  color: var(--gray-five);
}
</style>
