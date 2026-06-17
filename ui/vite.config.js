import { sveltekit } from '@sveltejs/kit/vite';

const gsUrbitMode = process.env.GS_URBIT_MODE === 'true';
const devPanel = process.env.GS_DEV_PANEL === 'true';
const customHostname = process.env.GS_CUSTOM_HOSTNAME;
const gsVersion = process.env.GS_VERSION || 'dev';
const perigeeWasmUrl = process.env.GS_PERIGEE_WASM_URL || 'https://files.native.computer/wasm/perigee.wasm';
const perigeeWasmExecUrl = process.env.GS_PERIGEE_WASM_EXEC_URL || 'https://files.native.computer/wasm/wasm_exec.js';
const config = {
	plugins: [sveltekit()],
  define: {
    'process.env.GS_URBIT_MODE': JSON.stringify(gsUrbitMode),
    'process.env.GS_DEV_PANEL': JSON.stringify(devPanel),
    'process.env.GS_CUSTOM_HOSTNAME': JSON.stringify(customHostname),
    'process.env.GS_PERIGEE_WASM_URL': JSON.stringify(perigeeWasmUrl),
    'process.env.GS_PERIGEE_WASM_EXEC_URL': JSON.stringify(perigeeWasmExecUrl),
    'import.meta.env.GS_VERSION': JSON.stringify(gsVersion),
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
      },
      '^/~groundseg/.*': {
        target: 'http://127.0.0.1:8082/',
        changeOrigin: true
      }
    },
    cors: true
  }
}

export default config;
