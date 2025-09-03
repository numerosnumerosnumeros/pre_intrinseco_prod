import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  envDir: '../build/',
  build: {
    outDir: '../build/dist',
    emptyOutDir: false,
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      '@views': fileURLToPath(new URL('./src/views', import.meta.url)),
      '@components': fileURLToPath(
        new URL('./src/components', import.meta.url),
      ),
      '@icons': fileURLToPath(
        new URL('./src/components/icons', import.meta.url),
      ),
      '@utils': fileURLToPath(new URL('./src/utils', import.meta.url)),
      '@state': fileURLToPath(new URL('./src/utils/state', import.meta.url)),
      '@config': fileURLToPath(new URL('./src/config', import.meta.url)),
    },
  },
})
