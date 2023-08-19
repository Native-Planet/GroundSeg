<script>
  import { structure,
    toggleDevMode,
    toggleAutoBoot,
    toggleNetwork,
    toggleUrbitPower
  } from '$lib/stores/websocket'

  import Power from './Section/Power.svelte'
  import CustomUrbitDomain from './Section/CustomUrbitDomain.svelte'
  import CustomMinIODomain from './Section/CustomMinIODomain.svelte'
  import MinIO from './Section/MinIO.svelte'
  import Loom from './Section/Loom.svelte'
  import PackMeld from './Section/PackMeld.svelte'
  import DevMode from './Section/DevMode.svelte'
  import RemoteAccess from './Section/RemoteAccess.svelte'

  import BottomPanel from './BottomPanel.svelte'

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
  <Power {running} {tTogglePower} on:click={()=>toggleUrbitPower(patp)} />

  <!-- Custom Urbit Domain -->
  <CustomUrbitDomain />

  <!-- Custom MinIO Domain -->
  <CustomMinIODomain />

  <!-- MinIO Settings -->
  <MinIO />

  <!-- Loom -->
  <Loom />

  <!-- Pack & Meld -->
  <PackMeld />

  <!-- Dev Mode -->
  <DevMode {devMode} on:click={()=>toggleDevMode(patp)} />

  <!-- Remote Access -->
  <RemoteAccess {remote} on:click={()=>toggleNetwork(patp)} />

  <!-- Bottom Panel -->
  <BottomPanel />

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
</style>
