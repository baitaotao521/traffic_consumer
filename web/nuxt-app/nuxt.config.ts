import { defineNuxtConfig } from 'nuxt/config';

export default defineNuxtConfig({
  ssr: false,
  app: {
    baseURL: '/',
    buildAssetsDir: '/static/nuxt/_nuxt/',
    head: {
      title: 'Traffic Consumer',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: '流量消耗器控制台' }
      ]
    }
  },
  css: [
    '@/assets/styles/main.scss'
  ],
  modules: ['@nuxt/ui'],
  nitro: {
    output: {
      // 将构建产物写入后端可托管目录
      publicDir: '../../static/nuxt'
    }
  },
  runtimeConfig: {
    public: {
      socketUrl: ''
    }
  }
});
