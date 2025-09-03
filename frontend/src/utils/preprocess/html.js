// REMEMBER: This occupies the main thread hence blocks the UI
export function processHTMLText(text) {
  if (!text) return ''

  if (isMime(text)) {
    const htmlContent = extractHtml(text)
    return processContent(htmlContent)
  }

  return processContent(text)
}

function isMime(content) {
  const searchArea = content.substring(0, 8192).toLowerCase()
  return (
    searchArea.includes('mime-version: 1.0') &&
    searchArea.includes('content-type: multipart/')
  )
}

// Extract HTML from multipart MIME message
function extractHtml(content) {
  const boundaryMatch = content.match(/boundary="([^"]+)"/i)
  if (!boundaryMatch) return ''

  const boundary = boundaryMatch[1]
  const parts = content.split('--' + boundary)

  let result = ''
  for (const part of parts) {
    if (part.toLowerCase().includes('content-type: text/html')) {
      // Handle quoted-printable encoding
      if (
        part
          .toLowerCase()
          .includes('content-transfer-encoding: quoted-printable')
      ) {
        const bodyStart =
          part.indexOf('\r\n\r\n') !== -1
            ? part.indexOf('\r\n\r\n') + 4
            : part.indexOf('\n\n') !== -1
              ? part.indexOf('\n\n') + 2
              : 0

        const body = part.substring(bodyStart)
        result += decodeQuoted(body) + '\n'
      } else {
        // Extract content after headers
        const bodyStart =
          part.indexOf('\r\n\r\n') !== -1
            ? part.indexOf('\r\n\r\n') + 4
            : part.indexOf('\n\n') !== -1
              ? part.indexOf('\n\n') + 2
              : 0

        result += part.substring(bodyStart) + '\n'
      }
    }
  }

  return result
}

// Decode quoted-printable encoding
function decodeQuoted(input) {
  return input
    .replace(/=([0-9A-F]{2})/gi, (match, hex) => {
      return String.fromCharCode(parseInt(hex, 16))
    })
    .replace(/=\r\n/g, '')
    .replace(/=\n/g, '')
}

/**
 * Process HTML content to extract readable text
 */
function processContent(content) {
  if (!content) return ''

  let result = ''
  let inScript = false
  let inStyle = false
  let inTable = false

  const parser = new DOMParser()
  const doc = parser.parseFromString(content, 'text/html')

  // Clean DOM
  removeSkippableTags(doc)

  // Extract text with formatting
  result = extractFormattedText(doc.body, inScript, inStyle, inTable)

  // Post-processing for entities and cleanup
  result = postprocessing(result)

  return result
}

/**
 * Remove tags that should be skipped
 */
function removeSkippableTags(doc) {
  const skippableTags = [
    'img',
    'meta',
    'button',
    'input',
    'svg',
    'noscript',
    'iframe',
    'link',
    'head',
    'nav',
    'header',
    'footer',
    'object',
    'embed',
    'canvas',
    'map',
    'area',
    'param',
    'video',
    'audio',
    'track',
    'source',
    'select',
    'base',
    'br',
    'col',
    'embed',
    'hr',
    'img',
    'input',
    'link',
    'meta',
    'param',
    'source',
    'track',
    'wbr',
  ]

  skippableTags.forEach(tag => {
    const elements = doc.querySelectorAll(tag)
    elements.forEach(el => {
      if (el.parentNode) {
        el.parentNode.removeChild(el)
      }
    })
  })
}

/**
 * Extract text with special handling for tables and formatting
 */
function extractFormattedText(node, inScript, inStyle, inTable) {
  if (!node) return ''

  let result = ''

  // Handle element nodes
  if (node.nodeType === Node.ELEMENT_NODE) {
    const tagName = node.nodeName.toLowerCase()

    // Track state
    if (tagName === 'script') {
      inScript = true
    } else if (tagName === 'style') {
      inStyle = true
    } else if (tagName === 'table') {
      inTable = true
      result += '\n\nTable: '
    }

    // Handle tables specially
    if (inTable) {
      if (tagName === 'tr') {
        result += '\n'
      } else if (tagName === 'td' || tagName === 'th') {
        result += '  '
      }
    }

    // Process children if not in script or style
    if (!inScript && !inStyle) {
      for (const child of node.childNodes) {
        result += extractFormattedText(child, inScript, inStyle, inTable)
      }
    }

    // Reset state after closing tags
    if (tagName === 'script') {
      inScript = false
    } else if (tagName === 'style') {
      inStyle = false
    } else if (tagName === 'table') {
      inTable = false
      result += '\nEnd of table\n\n'
    }
  }
  // Handle text nodes
  else if (node.nodeType === Node.TEXT_NODE) {
    if (!inScript && !inStyle) {
      const text = node.textContent.trim()
      if (text) {
        result += text + ' '
      }
    }
  }

  return result
}

/**
 * Process HTML entities and clean the output
 */
function postprocessing(input) {
  if (!input) return ''

  // Step 1: Handle HTML entities
  const entityMap = {
    '&quot;': '"',
    '&apos;': "'",
    '&amp;': '&',
    '&lt;': '<',
    '&gt;': '>',
    '&#160;': ' ',
    '&nbsp;': ' ',
    '&nbsp;nbsp;': ' ',
    '&nbsp;&nbsp;': ' ',
    '&#8217;': "'",
  }

  let output = input

  // Replace common entities
  for (const [entity, replacement] of Object.entries(entityMap)) {
    output = output.replaceAll(entity, replacement)
  }

  // Handle numeric entities
  output = output.replace(/&#(\d+);/g, (match, numStr) => {
    const num = parseInt(numStr, 10)
    return String.fromCodePoint(num)
  })

  // Handle hex entities
  output = output.replace(/&#[xX]([0-9a-fA-F]+);/g, (match, hexStr) => {
    const num = parseInt(hexStr, 16)
    return String.fromCodePoint(num)
  })

  // Step 2: Clean output (handling tables and whitespace)
  return cleanOutput(output)
}

/**
 * Final cleanup for the output
 */
function cleanOutput(input) {
  if (!input) return ''

  const tableStart = 'Table:'
  const tableEnd = 'End of table'

  let output = ''
  let consecutiveNewlines = 0

  // Process table sections specially
  let pos = 0
  let lastPos = 0

  while ((pos = input.indexOf(tableStart, pos)) !== -1) {
    // Add content before the table
    appendCleanedSection(input.substring(lastPos, pos))

    // Find the end of the table
    const endPos = input.indexOf(tableEnd, pos)
    if (endPos === -1) break

    const tableSection = input.substring(pos, endPos + tableEnd.length)

    // Check if table has numbers
    const numberCount = countNumbers(tableSection)

    if (numberCount > 1) {
      appendCleanedSection(tableSection)
    }

    pos = endPos + tableEnd.length
    lastPos = pos
  }

  // Add remaining content
  if (lastPos < input.length) {
    appendCleanedSection(input.substring(lastPos))
  }

  function appendCleanedSection(section) {
    for (let i = 0; i < section.length; i++) {
      if (section[i] === '\n') {
        if (consecutiveNewlines < 2) {
          output += '\n'
          consecutiveNewlines++
        }
      } else {
        output += section[i]
        consecutiveNewlines = 0
      }
    }
  }

  return output
}

/**
 * Count numeric values in a string
 */
function countNumbers(text) {
  const numberRegex = /-?(\d+(\.\d*)?|\.\d+)/g
  const matches = text.match(numberRegex) || []
  return matches.length
}
