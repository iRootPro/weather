import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
  base: '/app/',
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      includeAssets: ['favicon.svg'],
      cleanupOutdatedCaches: true,
      manifest: {
        name: 'Метеостанция Армавир',
        short_name: 'Погода',
        description: 'Умная лента внимания метеостанции Армавир',
        lang: 'ru-RU',
        id: '/app/',
        start_url: '/app/',
        scope: '/app/',
        display: 'standalone',
        orientation: 'portrait-primary',
        categories: ['weather', 'utilities'],
        background_color: '#10141f',
        theme_color: '#10141f',
        icons: [
          { src: '/static/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/static/icon-192.png', sizes: '192x192', type: 'image/png', purpose: 'maskable' },
          { src: '/static/icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: '/static/icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' }
        ]
      },
      workbox: {
        navigateFallback: '/app/index.html',
        skipWaiting: true,
        clientsClaim: true,
        runtimeCaching: [
          {
            urlPattern: ({ url }) => url.pathname.startsWith('/api/dashboard/snapshot'),
            handler: 'NetworkFirst',
            options: {
              cacheName: 'dashboard-snapshot',
              networkTimeoutSeconds: 4,
              expiration: { maxEntries: 8, maxAgeSeconds: 60 * 60 }
            }
          },
          {
            urlPattern: ({ url }) => url.pathname.startsWith('/api/weather/chart'),
            handler: 'NetworkFirst',
            options: {
              cacheName: 'weather-chart-24h',
              networkTimeoutSeconds: 4,
              expiration: { maxEntries: 12, maxAgeSeconds: 20 * 60 }
            }
          }
        ]
      }
    })
  ],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://192.168.1.161:8080'
    }
  }
});
