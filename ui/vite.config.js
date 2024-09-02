import { sveltekit } from '@sveltejs/kit/vite';

const gsUrbitMode = process.env.GS_URBIT_MODE === 'true';
const devPanel = process.env.GS_DEV_PANEL === 'true';
const customHostname = process.env.GS_CUSTOM_HOSTNAME;
const config = {
	plugins: [sveltekit()],
  define: {
    'process.env.GS_URBIT_MODE': JSON.stringify(gsUrbitMode),
    'process.env.GS_DEV_PANEL': JSON.stringify(devPanel),
    'process.env.GS_CUSTOM_HOSTNAME': JSON.stringify(customHostname),
  }
};
if (gsUrbitMode) {
  config["server"] = {
    proxy: {
      '^/session.js': {
        target: 'http://127.0.0.1:8082/',
        changeOrigin: true
      },
      '^/spider/.*': {
        target: 'http://127.0.0.1:8082/',
        changeOrigin: true
      },
      '^/~/.*': {
        target: 'http://127.0.0.1:8082/',
        changeOrigin: true
      }
    },
    cors: true
  }
}

export default config;
