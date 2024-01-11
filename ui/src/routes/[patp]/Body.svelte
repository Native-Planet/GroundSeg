<script>
  import { afterUpdate } from 'svelte'
  import { 
    toggleDevMode,
    toggleAutoBoot,
    toggleNetwork,
    toggleUrbitPower,
    installGallseg,
    uninstallGallseg
  } from '$lib/stores/websocket'

  import { structure, URBIT_MODE } from '$lib/stores/data'

  import Power from './Section/Power.svelte'
  import Urbit from './Section/Urbit.svelte'
  import MinIO from './Section/MinIO.svelte'
  import Loom from './Section/Loom.svelte'
  import PackMeld from './Section/PackMeld.svelte'
  import DevMode from './Section/DevMode.svelte'
  import RemoteAccess from './Section/RemoteAccess.svelte'
  import Chop from './Section/Chop.svelte'
  //import Gallseg from './Section/Gallseg.svelte'
  import AdminLogin from './Section/AdminLogin.svelte'

  import BottomPanel from './BottomPanel.svelte'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'
  export let patp

  let ownShip = false

  afterUpdate(()=>{
    if ($URBIT_MODE) {
      ownShip = (window.ship == patp)
    }
  })

  // info
  $: ship = ($structure?.urbits?.[patp]?.info)
  $: devMode = (ship?.devMode) || false
  $: detectBootStatus = (ship?.detectBootStatus) || false
  $: remote = (ship?.remote) || false
  $: remoteReady = (ship?.remoteReady) || false
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
  $: gallseg = (ship?.gallseg)
  $: authLevel = ($structure?.auth_level) || "unauthorized"

  // transitions
  $: tShip = ($structure?.urbits?.[patp]?.transition) || {}
  $: tTogglePower = (tShip?.togglePower) || ""
  $: tToggleDevMode = (tShip?.toggleDevMode) || ""
  $: tToggleNetwork = (tShip?.toggleNetwork) || ""
  $: tToggleMinIOLink = (tShip?.toggleMinIOLink) || ""
  $: tGallseg = tShip?.gallseg || ""

  // profile > startram
  $: startramRegistered = ($structure?.profile?.startram?.info?.registered) || false
  $: startramRunning = ($structure?.profile?.startram?.info?.running) || false

  const handleGallseg = p => {
    if (gallseg) {
      uninstallGallseg(p)
    } else {
      installGallseg(p)
    }
  }

</script>
<div class="body">
  <!-- Power -->
  <Power
    {patp}
    {running}
    {detectBootStatus}
    {tTogglePower}
    {ownShip}
    on:click={()=>toggleUrbitPower(patp)} 
    />

  <!-- Urbit Info -->
  <Urbit
    {showUrbAlias}
    {urbitAlias}
    {remote}
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
    {ownShip}
    />

  <!-- Remote Access -->
  <RemoteAccess
    on:click={()=>toggleNetwork(patp)}
    {patp}
    {remoteReady}
    {remote}
    {tToggleNetwork}
    {ownShip}
    />

  <!-- Dev Mode -->
  <DevMode
    on:click={()=>toggleDevMode(patp)}
    {patp}
    {devMode}
    {tToggleDevMode}
    {ownShip}
    />

  <!-- Loom -->
  <Loom
    {patp}
    {loomSize} 
    {ownShip}
    />
  
  <!-- temporarily disabled
  {#if !$URBIT_MODE}
    <Gallseg {gallseg} {tGallseg} on:click={()=>handleGallseg(patp)} />
  {/if}

  {#if $URBIT_MODE && (authLevel != "authorized")}
    <AdminLogin />
  {/if}
  -->

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
    min-width: 700px;
    border-radius: 16px 0 120px 16px;
    color: var(--text-card-color);
    display: flex;
    flex-direction: column;
    gap: 45px;
    overflow-y: scroll;
  }
</style>
