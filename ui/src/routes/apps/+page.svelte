<script>
  import './apps.css'
  import { onDestroy, onMount } from 'svelte'
  import { openModal } from 'svelte-modals'
  import Fa from 'svelte-fa'
  import {
    faAddressCard,
    faCheck,
    faCircleNotch,
    faClock,
    faCopy,
    faDownload,
    faEye,
    faEyeSlash,
    faFileExport,
    faKey,
    faPaperPlane,
    faRotate,
    faShield,
    faTriangleExclamation,
    faWallet,
    faXmark
  } from '@fortawesome/free-solid-svg-icons'
  import { structure } from '$lib/stores/data'
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
    { id: 'ticket', label: 'Master Ticket' },
    { id: 'private-key', label: 'ETH PK' },
    { id: 'wallet', label: 'HW Wallet' }
  ]

  const sections = [
    { id: 'keyfile', label: 'Keyfile', icon: faDownload },
    { id: 'breach', label: 'Breach', icon: faShield },
    { id: 'sponsor', label: 'Sponsor', icon: faKey },
    { id: 'transfer', label: 'Transfer', icon: faAddressCard },
    { id: 'proxy', label: 'Proxy', icon: faWallet }
  ]

  const sponsorOps = [
    { id: 'escape', label: 'Escape' },
    { id: 'cancel-escape', label: 'Cancel' },
    { id: 'adopt', label: 'Adopt' }
  ]

  const proxyOps = [
    { id: 'set-management-proxy', label: 'Manage' },
    { id: 'set-spawn-proxy', label: 'Spawn' },
    { id: 'set-transfer-proxy', label: 'Transfer' }
  ]

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
  let code = ''
  let revealKeyfile = false
  let copied = false
  let pollTimer

  $: urbits = ($structure?.urbits) || {}
  $: localShips = Object.keys(urbits)
  $: localShipExists = localShips.includes(ship)
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

  onMount(() => {
    loadKeyPending()
    pollDueKeyPending(false)
    pollTimer = setInterval(() => pollDueKeyPending(false), 15000)
  })

  onDestroy(() => {
    if (pollTimer) clearInterval(pollTimer)
  })

  const selectLocalShip = patp => {
    ship = patp
    loadPoint()
  }

  const loadPoint = async () => {
    if (!ship.trim()) return
    pointLoading = true
    actionError = ''
    try {
      const response = await getPoint(ship)
      ship = response.ship
      point = response.point
    } catch (error) {
      actionError = error.message
      point = null
    } finally {
      pointLoading = false
    }
  }

  const operationPayload = operation => ({
    operation,
    ship,
    credentialType,
    ticket,
    privateKey,
    passphrase,
    seed,
    sponsor,
    adoptee,
    newOwner,
    reset: resetTransfer,
    proxy
  })

  const rememberTransaction = response => {
    if (response.pending) addKeyPending(response.pending)
    actionMessage = response.message || 'Transaction submitted to Roller.'
    if (response.exportSuggested) {
      activeSection = 'breach'
    }
  }

  const submitOperation = async operation => {
    if (!ship.trim()) {
      actionError = 'Enter a ship first.'
      return
    }
    if (!credentialReady) {
      actionError = 'Select and enter signing credentials first.'
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

  const submitWallet = async payload => {
    if (!window.ethereum) {
      throw new Error('No browser wallet provider found.')
    }
    const prepared = await prepareWalletOperation({ ...payload, address: walletAddress })
    const signature = await window.ethereum.request({
      method: 'personal_sign',
      params: [prepared.signingPayload, walletAddress]
    })
    return submitWalletOperation({ ...payload, address: walletAddress, signature })
  }

  const connectWallet = async () => {
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

  const handleKeyfile = async () => {
    actionLoading = 'keyfile'
    actionError = ''
    actionMessage = ''
    try {
      const response = await generateKeyfile({ ship, ticket, passphrase })
      keyfile = response.keyfile
      keyfileName = response.filename
      downloadText(response.filename, response.keyfile)
      actionMessage = 'Keyfile generated.'
    } catch (error) {
      actionError = error.message
    } finally {
      actionLoading = ''
    }
  }

  const handleCode = async () => {
    actionLoading = 'code'
    actionError = ''
    actionMessage = ''
    try {
      const response = await generateCode({ ship, ticket, passphrase, step: 0 })
      code = response.code
      actionMessage = 'Network code generated.'
    } catch (error) {
      actionError = error.message
    } finally {
      actionLoading = ''
    }
  }

  const copyKeyfile = async () => {
    if (!keyfile) return
    await navigator.clipboard.writeText(keyfile)
    copied = true
    setTimeout(() => copied = false, 1600)
  }

  const exportShip = () => {
    if (!localShipExists) return
    openModal(ExportModal, { patp: ship })
  }

  const pendingTitle = tx => `${tx.operation || 'transaction'} ${tx.ship || ''}`.trim()
</script>

<div class="keys-shell">
  <div class="keys-grid">
    <section class="identity-panel">
      <div class="panel-heading">
        <div>
          <div class="eyebrow">ROLLER PKI</div>
          <h1>KEYS</h1>
        </div>
        <button class="icon-button" on:click={loadPoint} disabled={pointLoading || !ship.trim()} title="Refresh point">
          <Fa icon={pointLoading ? faCircleNotch : faRotate} spin={pointLoading} size="1x" />
        </button>
      </div>

      <label class="field-label" for="ship">Ship</label>
      <div class="ship-row">
        <input id="ship" bind:value={ship} list="local-ships" placeholder="~sampel-palnet" on:change={loadPoint} />
        <button class="compact-button" on:click={loadPoint} disabled={pointLoading || !ship.trim()}>
          <Fa icon={faRotate} size="1x" />
          Load
        </button>
      </div>
      <datalist id="local-ships">
        {#each localShips as patp}
          <option value={patp}></option>
        {/each}
      </datalist>

      {#if localShips.length > 0}
        <div class="local-strip">
          {#each localShips.slice(0, 5) as patp}
            <button class:active={ship === patp} on:click={() => selectLocalShip(patp)}>{patp}</button>
          {/each}
        </div>
      {/if}

      <div class="segmented" role="tablist" aria-label="Credential type">
        {#each credentials as credential}
          <button class:active={credentialType === credential.id} on:click={() => credentialType = credential.id}>
            {credential.label}
          </button>
        {/each}
      </div>

      {#if credentialType === 'ticket'}
        <label class="field-label" for="ticket">Master Ticket</label>
        <input id="ticket" bind:value={ticket} type="password" placeholder="~sampel-sampel-sampel-sampel" />
      {:else if credentialType === 'private-key'}
        <label class="field-label" for="private-key">Ethereum Private Key</label>
        <input id="private-key" bind:value={privateKey} type="password" placeholder="0x..." />
      {:else}
        <button class="wallet-button" on:click={connectWallet}>
          <Fa icon={faWallet} size="1x" />
          {walletAddress ? 'Reconnect Wallet' : 'Connect Wallet'}
        </button>
        {#if walletAddress}
          <div class="mono-line">{walletAddress}</div>
        {/if}
        {#if walletStatus}
          <div class="muted-line">{walletStatus}</div>
        {/if}
      {/if}

      <label class="field-label" for="passphrase">Passphrase</label>
      <input id="passphrase" bind:value={passphrase} type="password" placeholder="Optional" />

      <div class="point-readout">
        <div>
          <span>Dominion</span>
          <strong>{point?.dominion || 'unknown'}</strong>
        </div>
        <div>
          <span>Life</span>
          <strong>{pointLife || '-'}</strong>
        </div>
        <div>
          <span>Rift</span>
          <strong>{pointRift || '-'}</strong>
        </div>
        <div>
          <span>Sponsor</span>
          <strong>{pointSponsor || '-'}</strong>
        </div>
      </div>

      {#if point}
        <div class="address-list">
          <div><span>Owner</span><code>{pointOwner || '-'}</code></div>
          <div><span>Manage</span><code>{pointManager || '-'}</code></div>
          <div><span>Transfer</span><code>{pointTransfer || '-'}</code></div>
          <div><span>Spawn</span><code>{pointSpawn || '-'}</code></div>
        </div>
      {/if}
    </section>

    <section class="operation-panel">
      <div class="section-tabs">
        {#each sections as section}
          <button class:active={activeSection === section.id} on:click={() => activeSection = section.id}>
            <Fa icon={section.icon} size="1x" />
            {section.label}
          </button>
        {/each}
      </div>

      {#if activeSection === 'keyfile'}
        <div class="operation-body">
          <div class="operation-title">Boot key material</div>
          <p class="operation-copy">Generate a keyfile from the current PKI state and the selected master ticket.</p>
          {#if credentialType !== 'ticket'}
            <div class="notice compact">
              <Fa icon={faTriangleExclamation} size="1x" />
              Keyfile generation currently requires the master ticket.
            </div>
          {/if}
          <div class="action-row">
            <button class="primary-action" disabled={actionLoading || credentialType !== 'ticket' || !ship || !ticket} on:click={handleKeyfile}>
              <Fa icon={actionLoading === 'keyfile' ? faCircleNotch : faDownload} spin={actionLoading === 'keyfile'} size="1x" />
              Download Keyfile
            </button>
            <button class="secondary-action" disabled={actionLoading || credentialType !== 'ticket' || !ship || !ticket} on:click={handleCode}>
              <Fa icon={actionLoading === 'code' ? faCircleNotch : faKey} spin={actionLoading === 'code'} size="1x" />
              Generate Code
            </button>
          </div>
          {#if keyfile}
            <div class="keyfile-box">
              <div class="keyfile-toolbar">
                <span>{keyfileName}</span>
                <div>
                  <button class="icon-button small" on:click={() => revealKeyfile = !revealKeyfile} title="Reveal keyfile">
                    <Fa icon={revealKeyfile ? faEyeSlash : faEye} size="1x" />
                  </button>
                  <button class="icon-button small" on:click={copyKeyfile} title="Copy keyfile">
                    <Fa icon={copied ? faCheck : faCopy} size="1x" />
                  </button>
                </div>
              </div>
              {#if revealKeyfile}
                <pre>{keyfile}</pre>
              {/if}
            </div>
          {/if}
          {#if code}
            <div class="code-line">+code: <code>{code}</code></div>
          {/if}
        </div>
      {:else if activeSection === 'breach'}
        <div class="operation-body">
          <div class="operation-title">Continuity breach</div>
          <p class="operation-copy">Submit new network keys through the Roller. Batches are not immediate, so completion is tracked below.</p>
          <div class="notice">
            <Fa icon={faTriangleExclamation} size="1x" />
            Export the ship before booting after a successful breach.
            {#if localShipExists}
              <button on:click={exportShip}>
                <Fa icon={faFileExport} size="1x" />
                Export
              </button>
            {/if}
          </div>
          {#if credentialType !== 'ticket'}
            <label class="field-label" for="seed">Network Key Seed</label>
            <input id="seed" bind:value={seed} placeholder="64 hex characters" />
          {/if}
          <button class="danger-action" disabled={actionLoading || !ship || !credentialReady || (credentialType !== 'ticket' && !seed)} on:click={() => submitOperation('breach')}>
            <Fa icon={actionLoading === 'breach' ? faCircleNotch : faPaperPlane} spin={actionLoading === 'breach'} size="1x" />
            Submit Breach
          </button>
        </div>
      {:else if activeSection === 'sponsor'}
        <div class="operation-body">
          <div class="operation-title">Sponsorship</div>
          <div class="segmented compact-tabs">
            {#each sponsorOps as op}
              <button class:active={sponsorOperation === op.id} on:click={() => sponsorOperation = op.id}>{op.label}</button>
            {/each}
          </div>
          {#if sponsorOperation === 'adopt'}
            <label class="field-label" for="adoptee">Adoptee</label>
            <input id="adoptee" bind:value={adoptee} placeholder="~sampel-palnet" />
          {:else}
            <label class="field-label" for="sponsor">Sponsor</label>
            <input id="sponsor" bind:value={sponsor} placeholder="~sampel" />
          {/if}
          <button class="primary-action" disabled={actionLoading || !ship || !credentialReady} on:click={() => submitOperation(sponsorOperation)}>
            <Fa icon={actionLoading === sponsorOperation ? faCircleNotch : faPaperPlane} spin={actionLoading === sponsorOperation} size="1x" />
            Submit
          </button>
        </div>
      {:else if activeSection === 'transfer'}
        <div class="operation-body">
          <div class="operation-title">Ownership transfer</div>
          <label class="field-label" for="new-owner">New Owner Address</label>
          <input id="new-owner" bind:value={newOwner} placeholder="0x..." />
          <label class="check-row">
            <input type="checkbox" bind:checked={resetTransfer} />
            Reset transfer state
          </label>
          <button class="primary-action" disabled={actionLoading || !ship || !credentialReady || !newOwner} on:click={() => submitOperation('transfer')}>
            <Fa icon={actionLoading === 'transfer' ? faCircleNotch : faPaperPlane} spin={actionLoading === 'transfer'} size="1x" />
            Submit Transfer
          </button>
        </div>
      {:else if activeSection === 'proxy'}
        <div class="operation-body">
          <div class="operation-title">Proxy control</div>
          <div class="segmented compact-tabs">
            {#each proxyOps as op}
              <button class:active={proxyOperation === op.id} on:click={() => proxyOperation = op.id}>{op.label}</button>
            {/each}
          </div>
          <label class="field-label" for="proxy">Proxy Address</label>
          <input id="proxy" bind:value={proxy} placeholder="0x..." />
          <button class="primary-action" disabled={actionLoading || !ship || !credentialReady || !proxy} on:click={() => submitOperation(proxyOperation)}>
            <Fa icon={actionLoading === proxyOperation ? faCircleNotch : faPaperPlane} spin={actionLoading === proxyOperation} size="1x" />
            Set Proxy
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

  <section class="pending-band">
    <div class="pending-header">
      <div>
        <div class="eyebrow">ROLLUP QUEUE</div>
        <h2>Pending Transactions</h2>
      </div>
      <button class="compact-button" on:click={() => pollDueKeyPending(true)}>
        <Fa icon={faRotate} size="1x" />
        Check Now
      </button>
    </div>
    {#if $keyPending.length < 1}
      <div class="empty-pending">No saved Roller transactions.</div>
    {:else}
      <div class="pending-list">
        {#each $keyPending as tx}
          <div class="pending-item" class:complete={tx.status === 'complete' || tx.status === 'confirmed'}>
            <div class="pending-icon">
              <Fa icon={(tx.status === 'complete' || tx.status === 'confirmed') ? faCheck : faClock} size="1x" />
            </div>
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
    {/if}
  </section>
</div>
