import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@/components': path.resolve(__dirname, './src/components'),
      '@/pages': path.resolve(__dirname, './src/pages'),
      '@/api': path.resolve(__dirname, './src/api'),
      '@/stores': path.resolve(__dirname, './src/stores'),
      '@/types': path.resolve(__dirname, './src/types'),
      '@/lib': path.resolve(__dirname, './src/lib'),
      '@/hooks': path.resolve(__dirname, './src/hooks'),
      '@/config': path.resolve(__dirname, './src/config'),
      '@/layouts': path.resolve(__dirname, './src/layouts'),
      '@/utils': path.resolve(__dirname, './src/utils'),
    },
  },
  plugins: [
    tailwindcss(),
    react({
      babel: {
        plugins: [['babel-plugin-react-compiler', { target: '19' }]],
      },
    }),
  ],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // React 코어
          'vendor-react': ['react', 'react-dom'],
          // 라우팅
          'vendor-router': ['react-router-dom'],
          // 애니메이션
          'vendor-motion': ['framer-motion'],
          // 데이터 fetching
          'vendor-query': ['@tanstack/react-query'],
          // 아이콘
          'vendor-icons': ['lucide-react'],
          // 폼 & 유효성 검사
          'vendor-forms': ['react-hook-form', 'zod', '@hookform/resolvers'],
          // UI 프리미티브
          'vendor-ui': ['@headlessui/react', '@radix-ui/react-label', '@radix-ui/react-slot'],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/admin/api': {
        target: 'http://localhost:30001',
        changeOrigin: true,
      },
    },
  },
})
