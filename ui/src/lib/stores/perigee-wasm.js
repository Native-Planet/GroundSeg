const DEFAULT_WASM_URL = 'https://files.native.computer/wasm/perigee.wasm'
const DEFAULT_WASM_EXEC_URL = 'https://files.native.computer/wasm/wasm_exec.js'

const wasmUrl = process.env.GS_PERIGEE_WASM_URL || DEFAULT_WASM_URL
const wasmExecUrl = process.env.GS_PERIGEE_WASM_EXEC_URL || DEFAULT_WASM_EXEC_URL

let startup
let runtimeScript

const loadScript = src => {
  if (runtimeScript) return runtimeScript
  runtimeScript = new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[data-perigee-wasm-exec="${src}"]`)
    if (existing) {
      existing.addEventListener('load', resolve, { once: true })
      existing.addEventListener('error', reject, { once: true })
      if (window.Go) resolve()
      return
    }
    const script = document.createElement('script')
    script.src = src
    script.async = true
    script.dataset.perigeeWasmExec = src
    script.onload = resolve
    script.onerror = () => reject(new Error(`Failed to load Perigee WASM runtime: ${src}`))
    document.head.appendChild(script)
  })
  return runtimeScript
}

const waitForPerigee = async () => {
  const started = Date.now()
  while (!window.perigee) {
    if (Date.now() - started > 15000) {
      throw new Error('Timed out waiting for Perigee WASM exports')
    }
    await new Promise(resolve => setTimeout(resolve, 20))
  }
  return window.perigee
}

const rollerProxyURL = () => {
  const url = new URL('/~groundseg/roller', window.location.origin)
  return url.toString()
}

export const loadPerigeeWasm = async () => {
  if (startup) return startup
  startup = (async () => {
    if (typeof window === 'undefined' || typeof document === 'undefined') {
      throw new Error('Perigee WASM is only available in the browser')
    }
    await loadScript(wasmExecUrl)
    if (!window.Go) {
      throw new Error('Go WASM runtime did not initialize')
    }
    const go = new window.Go()
    const response = await fetch(wasmUrl, { cache: 'force-cache' })
    if (!response.ok) {
      throw new Error(`Failed to fetch Perigee WASM: ${response.status}`)
    }
    const bytes = await response.arrayBuffer()
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject)
    go.run(instance)
    const api = await waitForPerigee()
    if (api.setRollerProxy) {
      await api.setRollerProxy(rollerProxyURL())
    }
    return api
  })()
  return startup
}

export const callPerigee = async (method, payload = {}) => {
  const api = await loadPerigeeWasm()
  if (!api || typeof api[method] !== 'function') {
    throw new Error(`Perigee WASM method unavailable: ${method}`)
  }
  const result = await api[method](JSON.stringify(payload))
  const parsed = typeof result === 'string' ? JSON.parse(result) : result
  if (parsed?.ok === false) {
    throw new Error(parsed.error || `Perigee WASM ${method} failed`)
  }
  return parsed
}
