import { readFileSync } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const frontendRoot = path.resolve(__dirname, '..')
const repoRoot = path.resolve(frontendRoot, '..')
const publicDir = path.join(frontendRoot, 'public')

const env = {
  ...parseEnvFile(path.join(repoRoot, '.env')),
  ...parseEnvFile(path.join(repoRoot, '.env.local')),
  ...process.env,
}

const siteURL = normalizeSiteURL(env.VITE_SITE_URL || 'https://apigoqr.com')

await mkdir(publicDir, { recursive: true })

await writeFile(
  path.join(publicDir, 'robots.txt'),
  `User-agent: *\nAllow: /\n\nSitemap: ${siteURL}sitemap.xml\n`,
  'utf8',
)

await writeFile(
  path.join(publicDir, 'sitemap.xml'),
  `<?xml version="1.0" encoding="UTF-8"?>\n`
    + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n`
    + `  <url>\n`
    + `    <loc>${siteURL}</loc>\n`
    + `    <changefreq>weekly</changefreq>\n`
    + `    <priority>1.0</priority>\n`
    + `  </url>\n`
    + `</urlset>\n`,
  'utf8',
)

function normalizeSiteURL(value) {
  return value.replace(/\/+$/, '') + '/'
}

function parseEnvFile(filePath) {
  try {
    const content = readFileSync(filePath, 'utf8')
    const result = {}

    for (const rawLine of content.split(/\r?\n/)) {
      const line = rawLine.trim()
      if (!line || line.startsWith('#') || !line.includes('=')) {
        continue
      }

      const separatorIndex = line.indexOf('=')
      const key = line.slice(0, separatorIndex).trim()
      let value = line.slice(separatorIndex + 1).trim()
      if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
        value = value.slice(1, -1)
      }
      result[key] = value
    }

    return result
  } catch {
    return {}
  }
}
