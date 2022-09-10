//import adapter from '@sveltejs/adapter-auto';
import adapter from '@sveltejs/adapter-node';


/** @type {import('@sveltejs/kit').Config} */
const config = {
  kit: {
    prerender: {
      default: true
    },
    adapter: adapter({
    fallback: null
})}};

export default config;
