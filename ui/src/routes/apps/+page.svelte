<script>
  import './apps.css'
  import { onDestroy, onMount } from 'svelte'
  import { openModal } from 'svelte-modals'
  import Fa from 'svelte-fa'
  import {
    faCheck,
    faCircleNotch,
    faCopy,
    faDownload,
    faEye,
    faEyeSlash,
    faFileArrowUp,
    faFileExport,
    faPaperPlane,
    faRotate,
    faXmark
  } from '@fortawesome/free-solid-svg-icons'
  import { structure } from '$lib/stores/data'
  import { checkPatp, sigRemove } from '$lib/stores/patp'
  import { bootShip } from '$lib/stores/websocket'
  import {
    addKeyPending,
    downloadText,
    generateCode,
    generateKeyfile,
    getPoint,
    keyPending,
    loadKeyPending,
    pollDueKeyPending,
    prepareWalletOperation,
    removeKeyPending,
    submitKeyOperation,
    submitWalletOperation
  } from '$lib/stores/keys'
  import ExportModal from '../[patp]/ExportModal.svelte'

  const credentials = [
    { id: 'ticket', label: 'MASTER TICKET' },
    { id: 'private-key', label: 'ETH PK' },
    { id: 'wallet', label: 'HW WALLET' }
  ]

  const sections = [
    { id: 'keyfile', label: 'KEYFILE' },
    { id: 'breach', label: 'BREACH' },
    { id: 'sponsor', label: 'SPONSOR' },
    { id: 'transfer', label: 'TRANSFER' },
    { id: 'proxy', label: 'PROXY' }
  ]

  const sponsorOps = [
    { id: 'escape', label: 'ESCAPE' },
    { id: 'cancel-escape', label: 'CANCEL' },
    { id: 'adopt', label: 'ADOPT' }
  ]

  const proxyOps = [
    { id: 'set-management-proxy', label: 'MANAGE' },
    { id: 'set-spawn-proxy', label: 'SPAWN' },
    { id: 'set-transfer-proxy', label: 'TRANSFER' }
  ]

  const terminalStatuses = new Set(['complete', 'confirmed', 'failed'])

  let ship = ''
  let credentialType = 'ticket'
  let ticket = ''
  let privateKey = ''
  let passphrase = ''
  let seed = ''
  let walletAddress = ''
  let walletStatus = ''
  let activeSection = 'keyfile'
  let sponsorOperation = 'escape'
  let proxyOperation = 'set-management-proxy'
  let sponsor = ''
  let adoptee = ''
  let newOwner = ''
  let resetTransfer = false
  let proxy = ''
  let point = null
  let pointLoading = false
  let actionLoading = ''
  let actionError = ''
  let actionMessage = ''
  let keyfile = ''
  let keyfileName = ''
  let bootKeyfile = ''
  let bootFileName = ''
  let code = ''
  let revealKeyfile = false
  let copiedKeyfile = false
  let copiedCode = false
  let remote = true
  let selectedDrive = 'system-drive'
  let pollTimer

  $: urbits = ($structure?.urbits) || {}
  $: drives = ($structure?.system?.info?.drives) || {}
  $: driveNames = Object.keys(drives)
  $: startramReady = Boolean($structure?.profile?.startram?.info?.registered && $structure?.profile?.startram?.info?.running)
  $: localShips = Object.keys(urbits).filter(patp => !isMoon(patp)).sort()
  $: normalizedShip = normalizeShip(ship)
  $: localShipExists = Boolean(normalizedShip && urbits[normalizedShip])
  $: validShip = Boolean(normalizedShip && checkPatp(sigRemove(normalizedShip)))
  $: pointOwner = point?.ownership?.owner?.address || ''
  $: pointManager = point?.ownership?.managementProxy?.address || ''
  $: pointTransfer = point?.ownership?.transferProxy?.address || ''
  $: pointSpawn = point?.ownership?.spawnProxy?.address || ''
  $: pointLife = point?.network?.keys?.life || ''
  $: pointRift = point?.network?.rift || ''
  $: pointSponsor = point?.network?.sponsor?.patp || ''
  $: credentialReady = credentialType === 'ticket'
    ? ticket.trim().length > 0
    : credentialType === 'private-key'
      ? privateKey.trim().length > 0
      : walletAddress.trim().length > 0
  $: visiblePending = $keyPending.filter(tx => !terminalStatuses.has(tx.status))
  $: keyfileForBoot = (bootKeyfile || keyfile).trim()
  $: canBootFromKeyfile = validShip && !localShipExists && keyfileForBoot.length > 0

  onMount(() => {
    loadKeyPending()
    pollDueKeyPending(false)
    pollTimer = setInterval(() => pollDueKeyPending(false), 15000)
  })

  onDestroy(() => {
    if (pollTimer) clearInterval(pollTimer)
  })

  function normalizeShip(value) {
    const trimmed = value.trim()
    if (!trimmed) return ''
    return trimmed.startsWith('~') ? trimmed : `~${trimmed}`
  }

  function isMoon(patp) {
    return sigRemove(patp).split('-').length > 2
  }

  function randomSeed() {
    if (!globalThis.crypto?.getRandomValues) {
      throw new Error('Browser crypto is unavailable.')
    }
    const bytes = new Uint8Array(32)
    globalThis.crypto.getRandomValues(bytes)
    return Array.from(bytes, byte => byte.toString(16).padStart(2, '0')).join('')
  }

  function seedForOperation(operation) {
    if (operation !== 'breach' || credentialType === 'ticket') return ''
    if (!seed.trim()) seed = randomSeed()
    return seed.trim()
  }

  function operationPayload(operation) {
    return {
      operation,
      ship: normalizedShip,
      credentialType,
      ticket,
      privateKey,
      passphrase,
      seed: seedForOperation(operation),
      sponsor,
      adoptee,
      newOwner,
      reset: resetTransfer,
      proxy
    }
  }

  function operationReady(operation) {
    if (!validShip || !credentialReady || actionLoading) return false
    if (operation === 'escape' || operation === 'cancel-escape') return sponsor.trim().length > 0
    if (operation === 'adopt') return adoptee.trim().length > 0
    if (operation === 'transfer') return newOwner.trim().length > 0
    if (operation.startsWith('set-')) return proxy.trim().length > 0
    return true
  }

  async function loadPoint() {
    if (!validShip) {
      actionError = 'Enter a valid ship.'
      return
    }
    pointLoading = true
    actionError = ''
    try {
      const response = await getPoint(normalizedShip)
      ship = response.ship
      point = response.point
    } catch (error) {
      actionError = error.message
      point = null
    } finally {
      pointLoading = false
    }
  }

  function rememberTransaction(response) {
    if (response.pending) addKeyPending(response.pending)
    actionMessage = response.message || 'Transaction submitted to Roller.'
    if (response.exportSuggested) activeSection = 'breach'
  }

  async function submitOperation(operation) {
    if (!operationReady(operation)) {
      actionError = 'Complete the required fields first.'
      return
    }
    actionLoading = operation
    actionError = ''
    actionMessage = ''
    try {
      const payload = operationPayload(operation)
      const response = credentialType === 'wallet'
        ? await submitWallet(payload)
        : await submitKeyOperation(payload)
      rememberTransaction(response)
      await loadPoint()
    } catch (error) {
      actionError = error.message
    } finally {
      actionLoading = ''
    }
  }

  async function submitWallet(payload) {
    if (!window.ethereum) {
      throw new Error('No browser wallet provider found.')
    }
    const prepared = await prepareWalletOperation({ ...payload, address: walletAddress })
    if (prepared.seed && !payload.seed) payload.seed = prepared.seed
    const signature = await window.ethereum.request({
      method: 'personal_sign',
      params: [prepared.signingPayload, walletAddress]
    })
    return submitWalletOperation({ ...payload, address: walletAddress, signature })
  }

  async function connectWallet() {
    actionError = ''
    walletStatus = ''
    if (!window.ethereum) {
      walletStatus = 'No browser wallet provider found.'
      return
    }
    try {
      const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' })
      walletAddress = accounts?.[0] || ''
      credentialType = 'wallet'
      walletStatus = walletAddress ? 'Connected' : 'No account selected.'
    } catch (error) {
      walletStatus = error.message
    }
  }

  async function handleKeyfile() {
    actionLoading = 'keyfile'
    actionError = ''
    actionMessage = ''
    try {
      if (!validShip || !ticket.trim()) throw new Error('Enter a ship and master ticket.')
      const response = await generateKeyfile({ ship: normalizedShip, ticket, passphrase })
      keyfile = response.keyfile
      bootKeyfile = response.keyfile
      keyfileName = response.filename
      actionMessage = 'Keyfile generated.'
    } catch (error) {
      actionError = error.message
    } finally {
      actionLoading = ''
    }
  }

  async function handleCode() {
    actionLoading = 'code'
    actionError = ''
    actionMessage = ''
    try {
      if (!validShip || !ticket.trim()) throw new Error('Enter a ship and master ticket.')
      const response = await generateCode({ ship: normalizedShip, ticket, passphrase, step: 0 })
      code = response.code
      actionMessage = 'Network code generated.'
    } catch (error) {
      actionError = error.message
    } finally {
      actionLoading = ''
    }
  }

  function downloadKeyfile() {
    if (!keyfile) return
    downloadText(keyfileName || `${sigRemove(normalizedShip)}.key`, keyfile)
  }

  async function copyKeyfile() {
    if (!keyfile) return
    await navigator.clipboard.writeText(keyfile)
    copiedKeyfile = true
    setTimeout(() => copiedKeyfile = false, 1600)
  }

  async function copyCode() {
    if (!code) return
    await navigator.clipboard.writeText(code)
    copiedCode = true
    setTimeout(() => copiedCode = false, 1600)
  }

  async function handleKeyfileUpload(event) {
    const file = event.currentTarget.files?.[0]
    if (!file) return
    bootFileName = file.name
    bootKeyfile = await file.text()
    const fromName = file.name.replace(/\.key$/i, '')
    if (!ship.trim() && fromName) ship = normalizeShip(fromName)
  }

  function handleBootFromKeyfile() {
    actionError = ''
    actionMessage = ''
    if (localShipExists) {
      actionError = 'This ship is already present on this device.'
      return
    }
    if (!canBootFromKeyfile) {
      actionError = 'Enter a valid ship and keyfile.'
      return
    }
    bootShip(sigRemove(normalizedShip), keyfileForBoot, 'keyfile', startramReady ? remote : false, selectedDrive, '')
    actionMessage = `Boot requested for ${normalizedShip}.`
  }

  function exportShip() {
    if (!localShipExists) return
    openModal(ExportModal, { patp: normalizedShip })
  }

  function pendingTitle(tx) {
    return `${tx.operation || 'transaction'} ${tx.ship || ''}`.trim()
  }
</script>

<div class="keys-shell">
  <header class="keys-header">
    <div>
      <div class="eyebrow">ROLLER PKI</div>
      <h1>KEYS</h1>
    </div>
    <div class="endpoint">roller.urbit.org</div>
  </header>

  <div class="keys-grid">
    <section class="panel identity-panel">
      <div class="panel-title-row">
        <h2>POINT</h2>
        <button class="icon-button" on:click={loadPoint} disabled={pointLoading || !validShip} title="Refresh point">
          <Fa icon={pointLoading ? faCircleNotch : faRotate} spin={pointLoading} size="1x" />
        </button>
      </div>

      <label class="field-label" for="ship">SHIP</label>
      <div class="inline-row">
        <input id="ship" bind:value={ship} list="azimuth-ships" placeholder="~sampel-palnet" on:change={loadPoint} />
        <button class="btn secondary" on:click={loadPoint} disabled={pointLoading || !validShip}>LOAD</button>
      </div>
      <datalist id="azimuth-ships">
        {#each localShips as patp}
          <option value={patp}></option>
        {/each}
      </datalist>

      <div class="segmented" role="tablist" aria-label="Signing credential">
        {#each credentials as credential}
          <button class:active={credentialType === credential.id} on:click={() => credentialType = credential.id}>
            {credential.label}
          </button>
        {/each}
      </div>

      {#if credentialType === 'ticket'}
        <label class="field-label" for="ticket">MASTER TICKET</label>
        <input id="ticket" bind:value={ticket} type="password" placeholder="~sampel-sampel-sampel-sampel" />
      {:else if credentialType === 'private-key'}
        <label class="field-label" for="private-key">ETHEREUM PRIVATE KEY</label>
        <input id="private-key" bind:value={privateKey} type="password" placeholder="0x..." />
      {:else}
        <div class="field-label">EXTERNAL WALLET</div>
        <button class="btn secondary full" on:click={connectWallet}>{walletAddress ? 'RECONNECT WALLET' : 'CONNECT WALLET'}</button>
        {#if walletAddress}
          <code class="address-line">{walletAddress}</code>
        {/if}
        {#if walletStatus}
          <div class="status-text">{walletStatus}</div>
        {/if}
      {/if}

      <label class="field-label" for="passphrase">PASSPHRASE</label>
      <input id="passphrase" bind:value={passphrase} type="password" placeholder="optional" />

      {#if point}
        <div class="state-list">
          <div><span>DOMINION</span><strong>{point?.dominion || '-'}</strong></div>
          <div><span>LIFE</span><strong>{pointLife || '-'}</strong></div>
          <div><span>RIFT</span><strong>{pointRift || '-'}</strong></div>
          <div><span>SPONSOR</span><strong>{pointSponsor || '-'}</strong></div>
          <div><span>OWNER</span><code>{pointOwner || '-'}</code></div>
          <div><span>MANAGEMENT</span><code>{pointManager || '-'}</code></div>
          <div><span>TRANSFER</span><code>{pointTransfer || '-'}</code></div>
          <div><span>SPAWN</span><code>{pointSpawn || '-'}</code></div>
        </div>
      {/if}
    </section>

    <section class="panel operation-panel">
      <div class="operation-tabs" role="tablist" aria-label="PKI operation">
        {#each sections as section}
          <button class:active={activeSection === section.id} on:click={() => activeSection = section.id}>
            {section.label}
          </button>
        {/each}
      </div>

      {#if activeSection === 'keyfile'}
        <div class="operation-body">
          <div class="operation-heading">KEYFILE</div>
          <div class="button-row">
            <button class="btn primary" disabled={actionLoading || credentialType !== 'ticket' || !validShip || !ticket.trim()} on:click={handleKeyfile}>
              <Fa icon={actionLoading === 'keyfile' ? faCircleNotch : faDownload} spin={actionLoading === 'keyfile'} size="1x" />
              GENERATE
            </button>
            <button class="btn secondary" disabled={actionLoading || credentialType !== 'ticket' || !validShip || !ticket.trim()} on:click={handleCode}>
              <Fa icon={actionLoading === 'code' ? faCircleNotch : faPaperPlane} spin={actionLoading === 'code'} size="1x" />
              +CODE
            </button>
          </div>

          {#if keyfile}
            <div class="output-block">
              <div class="output-toolbar">
                <code>{keyfileName}</code>
                <div class="small-actions">
                  <button class="icon-button small" on:click={downloadKeyfile} title="Download keyfile">
                    <Fa icon={faDownload} size="1x" />
                  </button>
                  <button class="icon-button small" on:click={() => revealKeyfile = !revealKeyfile} title="Reveal keyfile">
                    <Fa icon={revealKeyfile ? faEyeSlash : faEye} size="1x" />
                  </button>
                  <button class="icon-button small" on:click={copyKeyfile} title="Copy keyfile">
                    <Fa icon={copiedKeyfile ? faCheck : faCopy} size="1x" />
                  </button>
                </div>
              </div>
              {#if revealKeyfile}
                <pre>{keyfile}</pre>
              {/if}
            </div>
          {/if}

          {#if code}
            <div class="code-row">
              <code>+code {code}</code>
              <button class="icon-button small" on:click={copyCode} title="Copy code">
                <Fa icon={copiedCode ? faCheck : faCopy} size="1x" />
              </button>
            </div>
          {/if}

          {#if normalizedShip && !localShipExists}
            <div class="boot-block">
              <div class="operation-heading small-heading">BOOT FROM KEYFILE</div>
              <label class="field-label" for="boot-keyfile">KEYFILE</label>
              <textarea id="boot-keyfile" bind:value={bootKeyfile} placeholder="paste keyfile"></textarea>
              <div class="button-row">
                <label class="file-button" for="keyfile-upload">
                  <Fa icon={faFileArrowUp} size="1x" />
                  UPLOAD
                </label>
                <input id="keyfile-upload" class="file-input" type="file" accept=".key,text/plain" on:change={handleKeyfileUpload} />
                <button class="btn primary" disabled={!canBootFromKeyfile} on:click={handleBootFromKeyfile}>
                  <Fa icon={faPaperPlane} size="1x" />
                  BOOT
                </button>
              </div>
              {#if bootFileName}
                <div class="status-text">{bootFileName}</div>
              {/if}
              {#if driveNames.length > 0 || startramReady}
                <div class="boot-options">
                  <label class="field-label" for="drive">DRIVE</label>
                  <select id="drive" bind:value={selectedDrive}>
                    <option value="system-drive">System Drive</option>
                    {#each driveNames as name}
                      <option value={name}>{drives[name].driveID == 0 ? 'New Drive' : `Drive ${drives[name].driveID}`} ({name})</option>
                    {/each}
                  </select>
                  {#if startramReady}
                    <label class="check-row">
                      <input type="checkbox" bind:checked={remote} />
                      <span>SET REMOTE</span>
                    </label>
                  {/if}
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {:else if activeSection === 'breach'}
        <div class="operation-body">
          <div class="operation-heading">BREACH</div>
          <div class="notice">
            <span>Export the ship before booting from breached keys.</span>
            {#if localShipExists}
              <button class="btn secondary" on:click={exportShip}>
                <Fa icon={faFileExport} size="1x" />
                EXPORT
              </button>
            {/if}
          </div>
          {#if credentialType !== 'ticket'}
            <details class="advanced">
              <summary>SEED OVERRIDE</summary>
              <label class="field-label" for="seed">NETWORK KEY SEED</label>
              <input id="seed" bind:value={seed} placeholder="optional 64 hex seed" />
            </details>
          {/if}
          <button class="btn danger" disabled={!operationReady('breach')} on:click={() => submitOperation('breach')}>
            <Fa icon={actionLoading === 'breach' ? faCircleNotch : faPaperPlane} spin={actionLoading === 'breach'} size="1x" />
            SUBMIT BREACH
          </button>
        </div>
      {:else if activeSection === 'sponsor'}
        <div class="operation-body">
          <div class="operation-heading">SPONSOR</div>
          <div class="segmented compact">
            {#each sponsorOps as op}
              <button class:active={sponsorOperation === op.id} on:click={() => sponsorOperation = op.id}>{op.label}</button>
            {/each}
          </div>
          {#if sponsorOperation === 'adopt'}
            <label class="field-label" for="adoptee">ADOPTEE</label>
            <input id="adoptee" bind:value={adoptee} placeholder="~sampel-palnet" />
          {:else}
            <label class="field-label" for="sponsor">SPONSOR</label>
            <input id="sponsor" bind:value={sponsor} placeholder="~sampel" />
          {/if}
          <button class="btn primary" disabled={!operationReady(sponsorOperation)} on:click={() => submitOperation(sponsorOperation)}>
            <Fa icon={actionLoading === sponsorOperation ? faCircleNotch : faPaperPlane} spin={actionLoading === sponsorOperation} size="1x" />
            SUBMIT
          </button>
        </div>
      {:else if activeSection === 'transfer'}
        <div class="operation-body">
          <div class="operation-heading">TRANSFER</div>
          <label class="field-label" for="new-owner">NEW OWNER</label>
          <input id="new-owner" bind:value={newOwner} placeholder="0x..." />
          <label class="check-row">
            <input type="checkbox" bind:checked={resetTransfer} />
            <span>RESET TRANSFER STATE</span>
          </label>
          <button class="btn primary" disabled={!operationReady('transfer')} on:click={() => submitOperation('transfer')}>
            <Fa icon={actionLoading === 'transfer' ? faCircleNotch : faPaperPlane} spin={actionLoading === 'transfer'} size="1x" />
            SUBMIT TRANSFER
          </button>
        </div>
      {:else if activeSection === 'proxy'}
        <div class="operation-body">
          <div class="operation-heading">PROXY</div>
          <div class="segmented compact">
            {#each proxyOps as op}
              <button class:active={proxyOperation === op.id} on:click={() => proxyOperation = op.id}>{op.label}</button>
            {/each}
          </div>
          <label class="field-label" for="proxy">ADDRESS</label>
          <input id="proxy" bind:value={proxy} placeholder="0x..." />
          <button class="btn primary" disabled={!operationReady(proxyOperation)} on:click={() => submitOperation(proxyOperation)}>
            <Fa icon={actionLoading === proxyOperation ? faCircleNotch : faPaperPlane} spin={actionLoading === proxyOperation} size="1x" />
            SET PROXY
          </button>
        </div>
      {/if}

      {#if actionError}
        <div class="error-line">{actionError}</div>
      {/if}
      {#if actionMessage}
        <div class="success-line">{actionMessage}</div>
      {/if}
    </section>
  </div>

  {#if visiblePending.length > 0}
    <section class="panel pending-panel">
      <div class="panel-title-row">
        <h2>PENDING</h2>
        <button class="btn secondary" on:click={() => pollDueKeyPending(true)}>
          <Fa icon={faRotate} size="1x" />
          CHECK
        </button>
      </div>
      <div class="pending-list">
        {#each visiblePending as tx}
          <div class="pending-item">
            <div class="pending-main">
              <strong>{pendingTitle(tx)}</strong>
              <code>{tx.hash || tx.signature || 'queued'}</code>
              {#if tx.lastError}
                <span>{tx.lastError}</span>
              {/if}
            </div>
            <div class="pending-status">{tx.status || 'pending'}</div>
            <button class="icon-button small" on:click={() => removeKeyPending(tx)} title="Remove saved transaction">
              <Fa icon={faXmark} size="1x" />
            </button>
          </div>
        {/each}
      </div>
    </section>
  {/if}
</div>
