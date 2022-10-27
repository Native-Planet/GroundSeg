<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

	import { updateState, api, system, isPortrait } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'

  import SysInfo from '$lib/SysInfo.svelte'
  import Power from '$lib/Power.svelte'

  import Network from '$lib/Network.svelte'
  import MinIO from '$lib/MinIO.svelte'
  import Contact from '$lib/Contact.svelte'

	// load data into store
	export let data
	updateState(data)

	let inViewSettings = true, tabs = ['Settings','Logs'], activeTab = 'Settings'

	// updateState loop
  const update = () => {
    if ($page.routeId == 'settings') {
			fetch($api + '/system')
			.then(raw => raw.json())
    	.then(res => updateState(res))
			.catch(err => console.log(err))

      console.log("settings query")
			setTimeout(update, 1000)
	}}

	// Start the update loop
	onMount(()=>update())

	// end the update loop
  onDestroy(()=> {console.log("settings destroy");inViewSettings = false})
	
</script>

{#if inViewSettings}
  <Card width="800px">
    <Logo t='System Settings'/>

    <!-- Settings Navigation -->
    <div class="navbar">
      {#each tabs as tab,i}
      <div 
        class="tab" 
        on:click={()=>activeTab = tab}
        class:active={tab == activeTab}
        transition:scale={{duration:120, delay: 200}}
        >
        {tab}
      </div>
      {/each}
    </div>

    {#if activeTab == 'Settings'}
      <div class="main-panel {$isPortrait ? "portrait" : "landscape"}">
        <div class="panel">
          <SysInfo
            ram={$system.ram} 
            temp={$system.temp}
            disk={$system.disk}
            cpu={$system.cpu}
            gsVersion={$system.gsVersion}
            updateMode={$system.updateMode}
            />
        </div>

        <div class="panel">
          <Network ethOnly={$system.ethOnly}/>
          <MinIO minio={$system.minio} />
        </div>
      </div>
      <div class="main-panel {$isPortrait ? "portrait" : "landscape"}">
        <div class="panel">
          <Power />
        </div>
        <div class="panel">
          <Contact />
        </div>
      </div>

    {/if}
  </Card>
{/if}

<style>
  .navbar {
    display: flex;
    margin: auto;
    margin-top: 12px;
    max-width: 360px;
    gap: 6px;
  }
  .tab {
    flex: 1;
    font-size: 14px;
    padding: 6px;
    text-align: center;
    border-radius: 8px;
    border: solid 1px #FFFFFF4D;
    cursor: pointer;
  }
  .tab:hover {background: #FFFFFF4D;}
  .active {
    background: var(--action-color);
    border-color: var(--action-color);
  }
  .active:hover {
    background: var(--action-color);
    opacity: .8;
  }
  .main-panel {
    margin-top: 12px;
    display: flex;
    gap: 12px;
  }
  .portrait { flex-direction: column;}
  .landscape { flex-direction: row;}
  .panel {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

</style>
