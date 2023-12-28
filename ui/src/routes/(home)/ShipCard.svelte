<script>
  import { structure, toggleNetwork } from '$lib/stores/websocket';

  import { devShipClass } from '$lib/stores/devclass'

  import Sigil from './Sigil.svelte'
  import StartramToggle from './StartramToggle.svelte'
  import NameBar from './NameBar.svelte'
  import ContainerInfo from './ContainerInfo.svelte'
  import ShipButtons from './ShipButtons.svelte'
  import { page } from '$app/stores'

  export let patp

  $: ship = ($structure?.urbits?.[patp]?.info) || {}
  $: running = (ship?.running) || false
  $: remote = (ship?.remote) || false
  $: remoteReady = (ship?.remoteReady) || false
  $: showUrbAlias = (ship?.showUrbAlias) || false
  $: urbitAlias = (ship?.urbitAlias) || ""

  $: url = (ship?.url) || "#"
  let urlType;
  $: {
    try {
      urlType = new URL(url);
    } catch (error) {
      urlType = null;
    }
  }
  $: urlStripped = urlType == null ? url : `${urlType?.hostname}`
  $: urlFixed = urlStripped == null ? url : remote ? url : (urlStripped == url) ? url : "http://" + $page.url.hostname + ":" + urlType.port 
  $: displayedUrl = (showUrbAlias && remote ? "https://"+urbitAlias : urlFixed)

  $: memUsage = (ship?.memUsage) || 0
  $: diskUsage = (ship?.diskUsage) || 0
  $: loom = (ship?.loomSize) || 0
  $: loomActual = 2 ** loom / (1024 * 1024)

  /* debug
  let on = false
  const loop = () => {on = !on;setTimeout(loop,3000)}
  loop()
  */

</script>

<div class="wrapper">
  <div class="sigil">
    <Sigil name={patp} />
  </div>
  <div class="bg"></div>
  <div class="toggle">
    <StartramToggle {remoteReady} on={remote} on:click={()=>toggleNetwork(patp)} />
    <!-- Debug
    <StartramToggle on={on} />
    -->
  </div>
  <div class="namebar">
    <NameBar {patp} {running} />
  </div>
  <div class="container-info">
    <ContainerInfo {memUsage} {diskUsage} loom={loomActual} />
  </div>
  <div class="buttons">
    <ShipButtons {patp} url={displayedUrl} />
  </div>
</div>

<style>
  .wrapper {
    position: relative;
    width: 320px;
    height: 180px;
    flex-shrink: 0;
  }
  .bg {
    position: absolute;
    width: 100%;
    height: 100%;
    background-image: url('/ship.svg');
    background-position: center;
    background-repeat: no-repeat;
    background-size: cover;
  }
  .sigil {
    position: absolute;
    background: var(--btn-secondary);
    width: 72px;
    height: 58px;
    flex-shrink: 0;
    left: 9px;
    border-radius: 8px 0;
    top: 6px;
  }
  .toggle {
    position: absolute;
    right: 0;
    top: 14px;
  }
  .namebar {
    position: absolute;
    top: 62px;
    left: 14px;
  }
  .container-info {
    position: absolute;
    bottom: 12px;
    left: 14px;
  }
  .buttons {
    position: absolute;
    right: 3px; 
    bottom: 2px;
  }
</style>
