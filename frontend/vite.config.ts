import { readFileSync } from 'node:fs'
import process from 'node:process'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

hydrateFrontendEnv()
const publicRoutes = readPublicRoutes()

export default defineConfig({
  envDir: '..',
  plugins: [vue(), tailwindcss()],
  ssgOptions: {
    script: 'async',
    formatting: 'minify',
    includedRoutes(paths) {
      const prerenderRoutes = new Set(['/', '/login', '/register', ...publicRoutes])
      return paths.filter((path) => prerenderRoutes.has(path))
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
  },
  build: {
    rollupOptions: {},
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './tests/setup.ts',
    include: ['tests/**/*.test.ts'],
    pool: 'forks',
    fileParallelism: false,
    maxWorkers: 1,
    execArgv: ['--max-old-space-size=4096'],
  },
})

function hydrateFrontendEnv() {
  const localEnv = {
    ...readEnvFile(fileURLToPath(new URL('../.env', import.meta.url))),
    ...readEnvFile(fileURLToPath(new URL('../.env.local', import.meta.url))),
  }

  for (const [key, value] of Object.entries(localEnv)) {
    if (!key.startsWith('VITE_')) {
      continue
    }
    process.env[key] = value
  }

  process.env.VITE_SITE_URL ??= process.env.VITE_API_BASE_URL ?? ''
}

function readEnvFile(filePath: string) {
  try {
    const content = readFileSync(filePath, 'utf8')
    const env: Record<string, string> = {}

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
      env[key] = value
    }

    return env
  } catch {
    return {}
  }
}

function readPublicRoutes() {
  const content = readFileSync(fileURLToPath(new URL('./src/content/public-pages.json', import.meta.url)), 'utf8')
  const pages = JSON.parse(content) as Array<{ path: string }>
  return pages.map((page) => page.path)
}
