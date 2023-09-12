<script>
  import { structure } from '$lib/stores/websocket';

  import { devShipClass } from '$lib/stores/devclass'

  import Sigil from './Sigil.svelte'
  import StartramToggle from './StartramToggle.svelte'
  import NameBar from './NameBar.svelte'
  import ContainerInfo from './ContainerInfo.svelte'
  import ShipButtons from './ShipButtons.svelte'

  export let patp
  let loom = 8192

  $: ship = ($structure?.urbits?.[patp]?.info) || {}
  $: running = (ship?.running) || false
  $: network = (ship?.network) || "none"
  $: url = (ship?.url) || "#"
  $: memUsage = (ship?.memUsage) || 0
  $: diskUsage = (ship?.diskUsage) || 0
  $: loom = (ship?.loomSize) || 0

  // debug
  let on = false
  const loop = () => {on = !on;setTimeout(loop,3000)}
  loop()
  // end debug

</script>

<div class="wrapper">
  <div class="sigil">
    <Sigil name={patp} />
  </div>
  <div class="bg"></div>
  <div class="toggle">
    <!--
    <StartramToggle on={network == "wireguard"} />
    -->
    <!-- Debug -->
    <StartramToggle on={on} />
  </div>
  <div class="namebar">
    <NameBar {patp} {running} />
  </div>
  <div class="container-info">
    <ContainerInfo {memUsage} {diskUsage} {loom} />
  </div>
  <div class="buttons">
    <ShipButtons {patp} {url}/>
  </div>
</div>

<style>
  .wrapper {
    position: relative;
    width: 288px;
    height: 148px;
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
    top: 51px;
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
