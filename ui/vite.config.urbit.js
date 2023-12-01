import { defineConfig } from 'vite';
import { sveltekit } from '@sveltejs/kit/vite';

export default ({ mode }) => {
  return defineConfig({
    plugins: [sveltekit()],
    server: {
      proxy: {
        '^/session.js': {
          target: 'http://127.0.0.1:8080/',
          changeOrigin: true
        },
        '^/spider/.*': {
          target: 'http://127.0.0.1:8080/',
          changeOrigin: true
        },
        '^/~/.*': {
          target: 'http://127.0.0.1:8080/',
          changeOrigin: true
        }
      },
      cors: true
    },
  });
};
