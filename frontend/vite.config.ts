import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  optimizeDeps: {
    include: ['nats.ws'],
  },
  server: {
    port: 5175,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true,
        // SSE support: keep connections alive and disable response buffering.
        configure: (proxy) => {
          proxy.on('proxyReq', (proxyReq) => {
            proxyReq.setHeader('Connection', 'keep-alive')
          })
          proxy.on('proxyRes', (proxyRes) => {
            if (proxyRes.headers['content-type']?.includes('text/event-stream')) {
              proxyRes.headers['x-accel-buffering'] = 'no'
              proxyRes.headers['cache-control'] = 'no-cache'
            }
          })
        },
      },
      '/assets': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
    },
  },
})
