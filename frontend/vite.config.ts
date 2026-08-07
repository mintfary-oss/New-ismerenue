import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],

  server: {
    port: 3000,
    proxy: {
      // В dev-режиме перенаправляем API-запросы на Go-бэкенд
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/widget': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },

  build: {
    outDir: 'dist',
    sourcemap: false,

    // Оптимизация для медленных соединений (3G ~1 Мбит/с = ~128 КБ/с)
    // Разбиваем на небольшие чанки — браузер кеширует каждый отдельно.
    // При повторном визите грузится только изменившийся чанк.
    rollupOptions: {
      output: {
        // Разбиваем на чанки — каждый кешируется браузером отдельно.
        // На 3G при повторном визите грузится только изменившийся чанк.
        manualChunks(id: string) {
          if (id.includes('node_modules/react') || id.includes('node_modules/react-dom') || id.includes('node_modules/react-router')) {
            return 'vendor-react';
          }
          if (id.includes('node_modules/recharts')) {
            return 'vendor-charts';
          }
          if (id.includes('node_modules/zustand')) {
            return 'vendor-state';
          }
        },
      },
    },

    // Предупреждение если чанк > 400 КБ (MapLibre исключён — он грузится лениво)
    chunkSizeWarningLimit: 400,

    // Минификация CSS (экономия ~20–30% трафика)
    cssMinify: true,

    // Целевые браузеры: современные мобильные + Chrome 80+/Firefox 75+/Safari 13+
    // Поддерживает устройства с Android 5+ (распространены в регионах с 3G)
    target: ['es2017', 'chrome80', 'firefox75', 'safari13'],
  },
});
