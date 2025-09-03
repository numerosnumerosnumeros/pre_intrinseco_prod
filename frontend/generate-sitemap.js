import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

const BASE_URL = 'https://nodo.finance'

const routes = [
  { url: '/', priority: 1.0, changefreq: 'weekly' },
  { url: '/terms-of-service', priority: 0.7, changefreq: 'monthly' },
  { url: '/documentation', priority: 0.8, changefreq: 'monthly' },
  { url: '/app', priority: 0.8, changefreq: 'weekly' },
  { url: '/app/company', priority: 0.6, changefreq: 'weekly' },
]

const sitemap = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  ${routes
    .map(
      route => `
  <url>
    <loc>${BASE_URL}${route.url}</loc>
    <lastmod>${new Date().toISOString().split('T')[0]}</lastmod>
    <changefreq>${route.changefreq}</changefreq>
    <priority>${route.priority}</priority>
  </url>`,
    )
    .join('')}
</urlset>`

// Write sitemap to public directory
fs.writeFileSync(path.resolve(__dirname, 'public/sitemap.xml'), sitemap.trim())

console.log('Sitemap generated successfully!')
