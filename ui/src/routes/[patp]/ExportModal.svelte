<script>
  import { 
    wsPort,
    exportUrbitShip,
    exportUrbitBucket
  } from '$lib/stores/websocket'
  import { loadSession } from '$lib/stores/gs-crypto'

  import { structure } from '$lib/stores/data'

  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  import { afterUpdate } from 'svelte'
  import { page } from '$app/stores'

  export let patp
  export let isOpen

  $: transition = ($structure?.urbits?.[patp]?.transition) || {}

  $: tExportShip = (transition?.exportShip) || ""
  $: tShipCompressed = (transition?.shipCompressed) || 0
  $: shipChanged = execIfShipChanged(tExportShip)

  $: tExportBucket = (transition?.exportBucket) || ""
  $: tBucketCompressed = (transition?.bucketCompressed) || 0
  $: bucketChanged = execIfBucketChanged(tExportBucket)

  $: startramRegistered = ($structure?.profile?.startram?.info?.registered) || false

  let shipExported = false
  let bucketExported = false

  const execIfShipChanged = async state => {
    if (state == "ready")
      await requestExport(patp)
    return state
  }

  const execIfBucketChanged = async state => {
    if (state == "ready")
      await requestExport("minio_"+patp)
    return state
  }

  const requestExport = async (container) => {
    // get token
    let token = await loadSession();
    if (!token.id || !token.token) {
      return;
    }
    // send request
    const hostname = $page.url.hostname
    const response = await fetch("http://"+hostname+":"+ $wsPort +"/export/"+container, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(token)
    });

    console.log(response)

    // handle response
    if (response.ok) {
      // Handle as Blob (file)
      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.style.display = 'none';
      a.href = url;
      // the filename you want
      a.download = container+'.zip';
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      if (container == patp) {
        shipExported = true
      } else {
        bucketExported = true
      }
    } else {
      console.log("Error:", response.status);
    }
  }
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <!-- debug
    <div>urbit: {JSON.stringify(tExportShip)}, storage: {JSON.stringify(tExportBucket)}</div>
    <div>tShipCompressed: {JSON.stringify(tShipCompressed)}</div>
    <div>tBucketCompressed: {JSON.stringify(tBucketCompressed)}</div>
    -->
    <div class="header">Export: <strong>{patp}</strong></div>
    <div class="name">Export Urbit Ship</div>
    <button
      disabled={(tExportShip != "") || shipExported}
      on:click={()=>exportUrbitShip(patp)}
      >
      {#if shipExported}
        Ship Exported
      {:else if tShipCompressed > 0}
        Compressing..{tShipCompressed}%
      {:else if tExportShip == "stopping"}
        Stopping Your Ship
      {:else if tExportShip == "ready"}
        Getting Zip File Ready
      {:else}
        Export Ship
      {/if}
    </button>
    <div class="name">Export Storage</div>
    <button
      disabled={(tExportBucket != "") || bucketExported}
      on:click={()=>exportUrbitBucket(patp)}
      >
      {#if bucketExported}
        Storage Exported
      {:else if tBucketCompressed > 0}
        Compressing..{tBucketCompressed}%
      {:else if tExportBucket == "stopping"}
        Stopping The Container
      {:else if tExportBucket == "ready"}
        Getting Zip File Ready
      {:else}
        Export Storage
      {/if}
    </button>
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
    margin: 32px 0 16px 0;
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
    height: 65px;
  }
  button:disabled {
    opacity: .6;
    pointer-events:none;
  }
</style>
