<script>
  import { onMount, tick } from 'svelte'
  import { get } from 'svelte/store'
  import { page } from '$app/stores'

  import Modal from '$lib/Modal.svelte'
  import { loadSession } from '$lib/stores/gs-crypto'
  import { URBIT_MODE } from '$lib/stores/data'
  import { wsPort } from '$lib/stores/websocket'

  import '@xterm/xterm/css/xterm.css'

  export let patp = ""
  export let target = "ship"
  export let title = "Web Shell"
  export let width = 980
  export let height = "60vh"
  export let minHeight = 420

  const customHostname = process.env.GS_CUSTOM_HOSTNAME

  let terminalEl
  let status = "Connecting..."
  let terminal
  let fitAddon
  let socket
  let resizeObserver
  let decoder = new TextDecoder()

  const sendMessage = message => {
    if (socket?.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(message))
    }
  }

  const resolveShellHost = () => {
    if (customHostname) {
      return customHostname
    }
    if (get(URBIT_MODE)) {
      return ""
    }
    return get(page).url.hostname
  }

  const writeStatusLine = message => {
    if (terminal) {
      terminal.writeln(`\r\n${message}`)
    }
  }

  const handleControlMessage = raw => {
    let message
    try {
      message = JSON.parse(raw)
    } catch {
      return
    }
    if (message.type === "ready") {
      status = "Connected"
      terminal?.focus()
      return
    }
    if (message.type === "error") {
      status = message.message || "Shell error"
      writeStatusLine(status)
      socket?.close()
      return
    }
    if (message.type === "exit") {
      status = Number.isInteger(message.code) ? `Session ended (${message.code})` : (message.message || "Session ended")
      writeStatusLine(status)
    }
  }

  const handleBinaryMessage = payload => {
    if (!terminal) {
      return
    }
    const bytes = payload instanceof Uint8Array ? payload : new Uint8Array(payload)
    const text = decoder.decode(bytes, { stream: true })
    if (text.length > 0) {
      terminal.write(text)
    }
  }

  onMount(() => {
    let disposed = false
    let dataDisposable
    let resizeDisposable

    ;(async () => {
      const shellHost = resolveShellHost()
      if (!shellHost) {
        status = "Set GS_CUSTOM_HOSTNAME to use the web shell in Urbit mode."
        return
      }

      const token = await loadSession()
      if (!token) {
        status = "Missing session token"
        return
      }

      const [{ Terminal }, { FitAddon }] = await Promise.all([
        import('@xterm/xterm'),
        import('@xterm/addon-fit'),
      ])
      if (disposed) {
        return
      }

      terminal = new Terminal({
        cursorBlink: true,
        convertEol: true,
        fontFamily: '"Source Code Pro", monospace',
        fontSize: 14,
        scrollback: 5000,
        theme: {
          background: '#161D17',
          foreground: '#DDE3DF',
          cursor: '#DDE3DF',
          selectionBackground: '#3E5142',
        },
      })
      fitAddon = new FitAddon()
      terminal.loadAddon(fitAddon)
      terminal.open(terminalEl)
      await tick()
      fitAddon.fit()
      terminal.focus()

      dataDisposable = terminal.onData(input => {
        sendMessage({ type: "input", input })
      })
      resizeDisposable = terminal.onResize(({ cols, rows }) => {
        sendMessage({ type: "resize", cols, rows })
      })

      resizeObserver = new ResizeObserver(() => {
        if (!terminal || !fitAddon) {
          return
        }
        fitAddon.fit()
        sendMessage({ type: "resize", cols: terminal.cols, rows: terminal.rows })
      })
      resizeObserver.observe(terminalEl)

      const protocol = get(page).url.protocol === "https:" ? "wss" : "ws"
      socket = new WebSocket(`${protocol}://${shellHost}:${get(wsPort)}/shell`)
      socket.binaryType = "arraybuffer"

      socket.onopen = () => {
        status = "Connecting shell..."
        sendMessage({
          patp,
          target,
          cols: terminal.cols,
          rows: terminal.rows,
          token,
        })
      }

      socket.onmessage = async event => {
        if (typeof event.data === "string") {
          handleControlMessage(event.data)
          return
        }
        if (event.data instanceof Blob) {
          handleBinaryMessage(await event.data.arrayBuffer())
          return
        }
        handleBinaryMessage(event.data)
      }

      socket.onerror = () => {
        if (status === "Connecting..." || status === "Connecting shell...") {
          status = "Web shell connection failed"
        }
      }

      socket.onclose = () => {
        if (
          status === "Connecting..." ||
          status === "Connecting shell..." ||
          status === "Connected"
        ) {
          status = "Disconnected"
        }
      }
    })()

    return () => {
      disposed = true
      resizeObserver?.disconnect()
      dataDisposable?.dispose?.()
      resizeDisposable?.dispose?.()
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(JSON.stringify({ type: "close" }))
      }
      socket?.close(1000)
      terminal?.dispose()
    }
  })
</script>

<Modal {width}>
  <div class="wrapper">
    <div class="header">
      <div class="title">{title}</div>
      <div class="status">{status}</div>
    </div>
    <div
      class="terminal"
      style={`--terminal-height: ${height}; --terminal-min-height: ${minHeight}px;`}
      bind:this={terminalEl}></div>
  </div>
</Modal>

<style>
  .wrapper {
    padding: 24px;
    width: calc(100% - 48px);
    color: var(--text-color);
  }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    margin-bottom: 12px;
  }
  .title {
    color: #000;
    font-family: Inter;
    font-size: 28px;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .status {
    color: var(--Gray-400, #5C7060);
    font-family: Inter;
    font-size: 16px;
    font-weight: 400;
    line-height: 22px;
    text-align: right;
  }
  .terminal {
    height: var(--terminal-height, 60vh);
    min-height: var(--terminal-min-height, 420px);
    border-radius: 16px;
    overflow: hidden;
    border: 1px solid #3E5142;
    background: #161D17;
    padding: 8px;
  }
  .terminal :global(.xterm) {
    height: 100%;
  }
  .terminal :global(.xterm-viewport) {
    overflow-y: auto;
  }
</style>
