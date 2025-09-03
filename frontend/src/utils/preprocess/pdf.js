export async function extractPDFText(client, file, pages = []) {
  const strToInt = (value, defaultVal = 1) => {
    if (value === undefined || value === null || value === '') return defaultVal

    const str = String(value).trim()

    const trimmed = str.replace(/^0+/, '')

    if (!trimmed || trimmed.length > 3 || !/^\d+$/.test(trimmed)) {
      return defaultVal
    }

    const val = parseInt(trimmed, 10)
    return val > 0 ? val : defaultVal
  }

  const minPageInt = strToInt(pages[0], 1)
  const maxPageInt = strToInt(pages[1], 1)

  const pageRange = {
    first: minPageInt,
    second: maxPageInt < minPageInt ? 1 : maxPageInt,
  }

  const fileURL = URL.createObjectURL(file)

  try {
    const loadingTask = client.getDocument({
      url: fileURL,
      verbosity: client.VerbosityLevel.ERRORS,
    })

    const pdf = await loadingTask.promise

    const MAX_PAGES = 100
    const docPages = pdf.numPages

    // Calculate start and end pages (zero-based in JS)
    let startPage = Math.max(0, pageRange.first - 1)
    let endPage

    if (pageRange.second > 1 && pageRange.second > pageRange.first) {
      // Specific range requested
      endPage = Math.min(docPages, pageRange.second)
    } else if (pageRange.first > 1 && pageRange.second <= 1) {
      // Only start page specified, extract up to MAX_PAGES from there
      endPage = Math.min(docPages, MAX_PAGES + startPage)
    } else {
      // Default case: extract up to MAX_PAGES from beginning
      endPage = Math.min(docPages, MAX_PAGES)
    }

    let fullText = ''

    // Process each page
    for (let i = startPage; i < endPage; i++) {
      const page = await pdf.getPage(i + 1)
      const textContent = await page.getTextContent()

      // Detect columns by analyzing x-coordinates
      const xPositions = textContent.items.map(item =>
        Math.round(item.transform[4]),
      )
      const xFrequency = {}
      xPositions.forEach(x => {
        xFrequency[x] = (xFrequency[x] || 0) + 1
      })

      // Find common x-positions that likely represent columns (with threshold)
      const columnThreshold = 2 // Minimum occurrences to be considered a column
      const columns = Object.keys(xFrequency)
        .filter(x => xFrequency[x] >= columnThreshold)
        .map(Number)
        .sort((a, b) => a - b)

      // Group by vertical position with tolerance
      const yTolerance = 5 // Allow items within 5 units to be considered the same line
      const lineGroups = {}

      textContent.items.forEach(item => {
        const y = Math.round(item.transform[5])

        // Find if there's a close enough existing line group
        let assigned = false
        for (const existingY in lineGroups) {
          if (Math.abs(y - existingY) <= yTolerance) {
            lineGroups[existingY].push(item)
            assigned = true
            break
          }
        }

        // Create new line group if needed
        if (!assigned) {
          lineGroups[y] = [item]
        }
      })

      // Sort lines from top to bottom
      const sortedYPositions = Object.keys(lineGroups).sort((a, b) => b - a)

      // For each line, organize by columns and join
      const lines = sortedYPositions.map(yPosition => {
        // Sort items by x position
        const lineItems = lineGroups[yPosition].sort(
          (a, b) => a.transform[4] - b.transform[4],
        )

        // Create an object with entries for each column
        const rowData = {}
        lineItems.forEach(item => {
          // Find which column this item belongs to
          const itemX = Math.round(item.transform[4])
          let columnIndex = columns.findIndex(x => Math.abs(x - itemX) < 20)
          if (columnIndex === -1) columnIndex = 0

          // Add to existing column text or create new
          if (!rowData[columnIndex]) rowData[columnIndex] = ''
          rowData[columnIndex] += item.str + ' '
        })

        // Join columns with tabs to maintain alignment
        return Object.values(rowData)
          .map(text => text.trim())
          .join('\t')
      })

      const pageText = lines.join('\n')
      fullText += pageText + '\n\n'
    }

    return fullText
  } catch (error) {
    console.error('Error extracting PDF text:', error)
    throw new Error(`Failed to extract text from PDF: ${error.message}`)
  } finally {
    URL.revokeObjectURL(fileURL) // Clean up
  }
}
