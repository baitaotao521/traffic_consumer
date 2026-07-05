import { defineConfig, type UserConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

type BuildOnLog = NonNullable<NonNullable<NonNullable<UserConfig['build']>['rolldownOptions']>['onLog']>
type BuildLog = Parameters<BuildOnLog>[1]

function isIgnoredVueUsePureAnnotation(log: BuildLog) {
  if (typeof log === 'string') return false
  const location = `${log.id || ''} ${log.loc?.file || ''} ${log.message}`
  return log.code === 'INVALID_ANNOTATION' && location.includes('@vueuse/core')
}

const handleBuildLog: BuildOnLog = (level, log, defaultHandler) => {
  if (level === 'warn' && isIgnoredVueUsePureAnnotation(log)) return
  defaultHandler(level, log)
}

// https://vite.dev/config/
export default defineConfig({
  base: '/static/app/',
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    outDir: '../static/app',
    emptyOutDir: true,
    cssCodeSplit: false,
    rolldownOptions: {
      onLog: handleBuildLog,
      output: {
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) return 'assets/app.css'
          return 'assets/[name][extname]'
        },
        chunkFileNames: 'assets/[name].js',
        codeSplitting: {
          groups: [
            {
              name: 'charts',
              test: /node_modules[\\/]@unovis[\\/]/,
              priority: 4,
            },
            {
              name: 'vue-vendor',
              test: /node_modules[\\/](@vue|@vueuse|vue)[\\/]/,
              priority: 3,
            },
            {
              name: 'data-vendor',
              test: /node_modules[\\/]@tanstack[\\/]/,
              priority: 2,
            },
            {
              name: 'ui-vendor',
              test: /node_modules[\\/](@lucide|class-variance-authority|clsx|reka-ui|tailwind-merge|vue-sonner)[\\/]/,
              priority: 1,
            },
            {
              name: 'vendor',
              test: /node_modules[\\/]/,
            },
          ],
        },
        entryFileNames: 'assets/app.js',
      },
    },
  },
})
