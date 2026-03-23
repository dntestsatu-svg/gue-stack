import { readFileSync } from 'node:fs'
import { mkdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath, URL } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const frontendRoot = path.resolve(__dirname, '..')
const repoRoot = path.resolve(frontendRoot, '..')
const publicDir = path.join(frontendRoot, 'public')
const publicPagesPath = path.join(frontendRoot, 'src', 'content', 'public-pages.json')

const env = {
  ...process.env,
  ...parseEnvFile(path.join(repoRoot, '.env')),
  ...parseEnvFile(path.join(repoRoot, '.env.local')),
}

const siteURL = normalizeSiteURL(env.VITE_SITE_URL || 'https://apigoqr.com')
const publicPages = JSON.parse(readFileSync(publicPagesPath, 'utf8'))
const today = new Date().toISOString().slice(0, 10)
const rootLastmod = await resolveLastmod([
  path.join(frontendRoot, 'src', 'views', 'LandingPageView.vue'),
  path.join(frontendRoot, 'src', 'style.css'),
])
const sitemapEntries = [
  {
    loc: siteURL,
    changefreq: 'weekly',
    priority: '1.0',
    lastmod: rootLastmod,
  },
  ...await Promise.all(publicPages.map(async (page) => ({
    loc: new URL(page.path, siteURL).toString(),
    changefreq: page.changefreq || 'monthly',
    priority: String(page.priority ?? 0.8),
    lastmod: await resolveLastmod([
      publicPagesPath,
      path.join(frontendRoot, 'src', 'views', 'PublicContentPageView.vue'),
      path.join(frontendRoot, 'src', 'style.css'),
    ]),
  }))),
]

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
    + sitemapEntries.map((entry) => (
      `  <url>\n`
      + `    <loc>${entry.loc}</loc>\n`
      + `    <lastmod>${entry.lastmod}</lastmod>\n`
      + `    <changefreq>${entry.changefreq}</changefreq>\n`
      + `    <priority>${entry.priority}</priority>\n`
      + `  </url>\n`
    )).join('')
    + `</urlset>\n`,
  'utf8',
)

function normalizeSiteURL(value) {
  return value.replace(/\/+$/, '') + '/'
}

async function resolveLastmod(filePaths) {
  const { stat } = await import('node:fs/promises')

  const timestamps = await Promise.all(filePaths.map(async (filePath) => {
    try {
      const info = await stat(filePath)
      return info.mtimeMs
    } catch {
      return 0
    }
  }))

  const latest = Math.max(...timestamps, Date.now())
  return new Date(latest).toISOString().slice(0, 10) || today
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
