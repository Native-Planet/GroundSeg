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
  $: loomSize = (ship?.loomSize)

</script>
<div class="body">
  <!-- Power -->
  <Power {running} {tTogglePower} on:click={()=>toggleUrbitPower(patp)} />

  <!-- Custom Urbit Domain -->
  <CustomUrbitDomain />

  <!-- Custom MinIO Domain -->
  <CustomMinIODomain />

  <!-- Loom -->
  <Loom {patp} {loomSize} />

  <!-- Pack & Meld -->
  <PackMeld />

  <!-- Dev Mode -->
  <DevMode {devMode} on:click={()=>toggleDevMode(patp)} />

  <!-- Remote Access -->
  <RemoteAccess {remote} on:click={()=>toggleNetwork(patp)} />

  <!-- Bottom Panel -->
  <BottomPanel {patp}/>
</div>

<style>
  .body::-webkit-scrollbar {display: none;}
  .body {
    background-color: var(--bg-card);
    position: absolute;
    bottom: 0;
    height: calc(743px - 150px - 20px);
    width: calc(100% - 40px);
    padding: 20px 20px 0 20px;
    max-width: 100vw;
    border-radius: 16px 0 120px 16px;
    color: var(--text-card-color);
    display: flex;
    flex-direction: column;
    gap: 45px;
    overflow-y: scroll;
  }
</style>
