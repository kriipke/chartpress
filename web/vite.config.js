import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In production the SPA is served same-origin with the API (an ingress routes
// /generate, /charts, /text-to-config to chartpress-server and / to the SPA).
// For `npm run dev`, proxy those API paths to a local server (default :8080,
// override with CHARTPRESS_API). See src/app/api.js.
const apiTarget = process.env.CHARTPRESS_API || 'http://localhost:8080'
const apiProxy = { target: apiTarget, changeOrigin: true }

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/',
  server: {
    proxy: {
      '/generate': apiProxy,
      '/charts': apiProxy,
      '/text-to-config': apiProxy,
    },
  },
})
