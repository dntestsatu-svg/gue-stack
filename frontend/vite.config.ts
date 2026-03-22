import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  envDir: '..',
  plugins: [vue(), tailwindcss()],
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
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) {
            return
          }

          if (
            id.includes('reka-ui')
            || id.includes('lucide-vue-next')
            || id.includes('@tabler/icons-vue')
            || id.includes('vue-sonner')
            || id.includes('class-variance-authority')
            || id.includes('tailwind-merge')
            || id.includes('clsx')
          ) {
            return 'vendor-ui'
          }

          if (id.includes('pinia') || id.includes('vue-router')) {
            return 'vendor-state'
          }

          if (id.includes('axios')) {
            return 'vendor-network'
          }

          if (id.includes('@internationalized/date')) {
            return 'vendor-date'
          }

          if (id.includes('/vue/') || id.includes('@vue')) {
            return 'vendor-vue'
          }

          return 'vendor-misc'
        },
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './tests/setup.ts',
    include: ['tests/**/*.test.ts'],
  },
})
