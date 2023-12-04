import adapter from '@sveltejs/adapter-static';

const gsUrbitMode = process.env.GS_URBIT_MODE === 'true';

const makeKit = () => {
  console.log("GS_URBIT_MODE: ",gsUrbitMode)
  let kit = {
    kit: {
      adapter: adapter({
        // default options are shown. On some platforms
        // these options are set automatically — see below
        pages: 'build',
        assets: 'build',
        fallback: 'index.html',
        precompress: false,
        strict: true
      }),
    }
  }
  kit.kit["paths"] = { 
    base: '/apps/groundseg',
  }
  return kit
}
const config = makeKit();
export default config
