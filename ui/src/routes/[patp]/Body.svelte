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
  $: loomSize = (ship?.loomSize)
  $: lusCode = "xxxxxx-xxxxxx-xxxxxx-xxxxxx"
  $: url = (ship?.url) || "#"

  $: tShip = ($structure?.urbits?.[patp]?.transition) || {}
  $: tTogglePower = (tShip?.togglePower) || ""
  $: tToggleDevMode = (tShip?.toggleDevMode) || ""
  $: tToggleNetwork = (tShip?.toggleNetwork) || ""
</script>
<div class="body">
  <!-- Power -->
  <Power {running} {tTogglePower} on:click={()=>toggleUrbitPower(patp)} />

  <!-- Custom Urbit Domain -->
  <CustomUrbitDomain {url} />

  <!-- Custom MinIO Domain -->
  <CustomMinIODomain />

  <!-- Loom -->
  <Loom {patp} {loomSize} />

  <!-- Pack & Meld -->
  <PackMeld />

  <!-- Dev Mode -->
  <DevMode {devMode} {tToggleDevMode} on:click={()=>toggleDevMode(patp)} />

  <!-- Remote Access -->
  <RemoteAccess {remote} {tToggleNetwork} on:click={()=>toggleNetwork(patp)} />

  <!-- Bottom Panel -->
  <BottomPanel {patp}/>
</div>

<style>
  .body::-webkit-scrollbar {display: none;}
  .body {
    background-color: var(--bg-card);
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
