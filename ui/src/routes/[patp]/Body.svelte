<script>
  import { structure,
    toggleDevMode,
    toggleAutoBoot,
    toggleNetwork,
    toggleUrbitPower
  } from '$lib/stores/websocket'

  import Power from './Section/Power.svelte'
  import Urbit from './Section/Urbit.svelte'
  import MinIO from './Section/MinIO.svelte'
  import Loom from './Section/Loom.svelte'
  import PackMeld from './Section/PackMeld.svelte'
  import DevMode from './Section/DevMode.svelte'
  import RemoteAccess from './Section/RemoteAccess.svelte'
  import Chop from './Section/Chop.svelte'

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
  $: lusCode = (ship?.lusCode) || ""
  $: url = (ship?.url) || "#"
  $: showUrbAlias = (ship?.showUrbAlias) || false
  $: urbitAlias = (ship?.urbitAlias) || ""
  $: minioAlias = (ship?.minioAlias) || ""
  $: minioUrl = (ship?.minioUrl) || "#"
  $: minioPwd = (ship?.minioPwd) || ""
  $: minioLinked = (ship?.minioLinked) || false

  $: tShip = ($structure?.urbits?.[patp]?.transition) || {}
  $: tTogglePower = (tShip?.togglePower) || ""
  $: tToggleDevMode = (tShip?.toggleDevMode) || ""
  $: tToggleNetwork = (tShip?.toggleNetwork) || ""
  $: tToggleMinIOLink = (tShip?.toggleMinIOLink) || ""

  $: startramRegistered = ($structure?.profile?.startram?.info?.registered) || false
  $: startramRunning = ($structure?.profile?.startram?.info?.running) || false
</script>
<div class="body">
  <!-- Power -->
  <Power
    {patp}
    {running}
    {detectBootStatus}
    {tTogglePower}
    on:click={()=>toggleUrbitPower(patp)} 
    />

  <!-- Urbit Info -->
  <Urbit
    {showUrbAlias}
    {urbitAlias}
    {url}
    {patp}
    {lusCode}
    {running}
    {startramRegistered}
    />

  {#if startramRegistered}
    <!-- MinIO Info -->
    <MinIO 
      {running}
      {startramRunning}
      {patp}
      {minioAlias}
      {minioUrl}
      {minioPwd}
      {minioLinked}
      {tToggleMinIOLink}
      />
  {/if}

  <!-- Pack & Meld -->
  <PackMeld
    {patp}
    />

  <!-- Remote Access -->
  <RemoteAccess {remote} {tToggleNetwork} on:click={()=>toggleNetwork(patp)} />

  <!-- Dev Mode -->
  <DevMode {devMode} {tToggleDevMode} on:click={()=>toggleDevMode(patp)} />

  <!-- Loom -->
  <Loom {patp} {loomSize} />

  <!-- Chop --
  <Chop
    {patp}
    />
  -->

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
