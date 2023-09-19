<script>
  import { 
    structure,
    exportUrbitShip,
    exportUrbitBucket
  } from '$lib/stores/websocket'
  import { page } from '$app/stores'
  import { loadSession } from '$lib/stores/gs-crypto'

  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  import { afterUpdate } from 'svelte'

  export let patp
  export let isOpen

  $: transition = ($structure?.urbits?.[patp]?.transition) || {}
  $: exportShip = (transition?.exportShip) || ""
  $: shipCompressed = (transition?.shipCompressed) || 0

  $: shipChanged = execIfChanged(exportShip)

  let shipExported = false
  let bucketExported = false

  const execIfChanged = async state => {
    if (state == "ready")
      await requestPier()
    return state
  }

  const requestPier = async () => {
    // get token
    let token = await loadSession();
    if (!token.id || !token.token) {
      return;
    }
    // send request
    const hostname = $page.url.hostname
    const port = "3000"
    const response = await fetch("http://"+hostname+":"+port+"/export/"+patp, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(token)
    });

    // handle response
    if (response.ok) {
      // Handle as Blob (file)
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.style.display = 'none';
      a.href = url;
      // the filename you want
      a.download = patp+'.zip';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      shipExported = true
    } else {
      console.log("Error:", response.status);
    }
  }
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <!-- debug --
    <div>urbit: {JSON.stringify(exportShip)}, storage: tbd</div>
    <div>shipCompressed: {JSON.stringify(shipCompressed)}</div>
    <!-- end debug -->

    <div class="header">Export For {patp}</div>
    <div class="name">What do you want to export?</div>
    <div class="button-wrapper">
      <button
        disabled={(exportShip != "") || shipExported}
        on:click={()=>exportUrbitShip(patp)}
        >
        {#if shipExported}
          Ship Exported
        {:else if shipCompressed > 0}
          Compressing..{shipCompressed}%
        {:else if exportShip == "stopping"}
          Stopping Your Ship
        {:else if exportShip == "ready"}
          Getting Zip File Ready
        {:else}
          Urbit Ship
        {/if}
      </button>
      <button
        on:click={()=>exportUrbitBucket(patp)}
        >
        Storage
      </button>
    </div>
  </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .header {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .name {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 42px; /* 133.333% */
    letter-spacing: -1.44px;
    max-width: 460px;
    margin: 32px 0;
  }
  .button-wrapper {
    display: flex;
    gap: 48px;
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: #000;
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events:none;
  }
</style>
