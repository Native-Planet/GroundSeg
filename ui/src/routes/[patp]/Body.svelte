<script>
  import { structure,
    toggleDevMode,
    toggleAutoBoot,
    toggleNetwork,
    toggleUrbitPower
  } from '$lib/stores/websocket'

  import { showDeleteModal } from './store'
  import Power from './Section/Power.svelte'
  import Loom from './Section/Loom.svelte'
  import DevMode from './Section/DevMode.svelte'
  import RemoteAccess from './Section/RemoteAccess.svelte'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'
  export let patp

  $: ship = ($structure?.urbits?.[patp]?.info)
  $: devMode = (ship?.devMode) || false
  $: detectBootStatus = (ship?.detectBootStatus) || false
  $: remote = (ship?.remote) || false
  $: running = (ship?.running) || false
  $: tShip = ($structure?.urbits?.[patp]?.transition) || {}
  $: tTogglePower = (tShip?.togglePower) || null

</script>
<div class="body">

  <!-- Power -->
  <Power
    {running}
    {tTogglePower}
    on:click={()=>toggleUrbitPower(patp)}
    />

  <!-- Custom Urbit Domain -->
  <div>minio custom domain</div>

  <!-- Custom MinIO Domain -->
  <div>urbit custom domain</div>

  <!-- Loom -->
  <Loom 
    />

  <!-- Pack & Meld -->
  <div class="section">
    <div class="section-left">
      <div class="title">Pack & Meld</div>
      <div class="description">
        This function will refragement your ship's memory capacity, optimizing its performance. We recommend scheduling these once a week
      </div>
    </div>
    <div class="section-right">
      Start Button, calendar
    </div>
  </div>

  <!-- Dev Mode -->
  <DevMode
    {devMode}
    on:click={()=>toggleDevMode(patp)}
    />

  <!-- Remote Access -->
  <RemoteAccess
    {remote}
    on:click={()=>toggleNetwork(patp)}
    />


  <div>MinIO</div>

  <div class="bottom-panel">
    <div class="btn">Logs</div>
    <div class="btn" on:click={()=>toggleUrbitPower(patp)}>
      {#if tTogglePower}
        {tTogglePower}
      {:else}
        {running ? "Shutdown" : "Boot"}
      {/if}
    </div>
    <div class="btn">Export</div>
    <div class="btn" on:click={()=>showDeleteModal.set(true)}>Delete</div>
  </div>
</div>

<style>
  .body {
    background-color: var(--bg-card);
    position: absolute;
    bottom: 0;
    height: calc(743px - 150px - 40px);
    width: calc(100% - 40px);
    padding: 20px;
    max-width: 100vw;
    border-radius: 16px 0 120px 16px;
    color: var(--text-card-color);
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .bottom-panel {
    position: absolute;
    bottom: 0;
    left: 32px;
    display: flex;
    gap: 8px;
  }
  .section {
    display: flex;
    align-items: center;
  }
  .section-left {
    flex: 1;
  }
  .section-right {
    flex: 1;
    text-align: right;
  }
  .btn {
    color: var(--text-card-color);
    font-size: 12px;
    background-color: var(--fg-card);
    border-radius: 16px 16px 0 0;
    padding: 12px 42px;
  }
  .title {
    font-size: 16px;
    font-family: var(--regular-font);
  }
</style>
