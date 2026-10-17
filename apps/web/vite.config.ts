import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_AUTH_URL || 'https://p01--pizza--4gxsm5p6nnbw.code.run',
        changeOrigin: true,
        secure: true,
      },
    },
  },
})
