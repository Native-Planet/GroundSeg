import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	kit: {
    prerender: {
      default: true
    },
		adapter: adapter({
      fallback: null
    })
	}
};

export default config;
