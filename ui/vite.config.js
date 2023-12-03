import { sveltekit } from '@sveltejs/kit/vite';

const gsUrbitMode = process.env.GS_URBIT_MODE === 'true';

const config = {
	plugins: [sveltekit()],
  define: {
    'process.env.GS_URBIT_MODE': JSON.stringify(gsUrbitMode),
  }
};

export default config;
