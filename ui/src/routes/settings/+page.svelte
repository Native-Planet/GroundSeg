<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'
  import { Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@rgossiaux/svelte-headlessui"

	import { updateState, api, system, isPortrait } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  import Logs from '$lib/Logs.svelte'
  import SysInfo from '$lib/SysInfo.svelte'
  import Power from '$lib/Power.svelte'

  import Network from '$lib/Network.svelte'
  import MinIO from '$lib/MinIO.svelte'
  import Sessions from '$lib/Sessions.svelte'
  import Contact from '$lib/Contact.svelte'

	// load data into store
	export let data
	updateState(data)

  let inViewSettings = false, 
    tabs = ['Settings','Logs'],
    activeTab = 'Settings',
    selectedContainer

	// updateState loop
  const update = () => {
    if (($page.routeId == 'settings') && (activeTab == 'Settings')) {
      fetch($api + '/system', {
        credentials: "include"
      })
			.then(raw => raw.json())
    	.then(res => updateState(res))
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

  const exportLogs = () => {
    let module = 'logs'
    fetch($api + '/system?module=' + module, {
		  method: 'POST',
      credentials: "include",
		  headers: {'Content-Type': 'application/json'},
  	  body: JSON.stringify({'action':'export','container':selectedContainer})
	  })
      .then(r => r.json())
      .then(d => {
          if (d == 404) {
            window.location.href = "/login"
          } else {
          var element = document.createElement('a')
          element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(d))
          element.setAttribute('download', selectedContainer)
          element.style.display = 'none'
          document.body.appendChild(element)
          element.click()
          document.body.removeChild(element)
          }
      })
  }

	// Start the update loop
  onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    if (data['status'] == 404) {
      window.location.href = "/login"
    }
    update()
    inViewSettings = true
    selectedContainer = $system.containers[0]
  })

	// end the update loop
  onDestroy(()=> inViewSettings = false)
	
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

        <div class="panel" in:scale={{duration:120, delay: 200}}>
          <SysInfo
            ram={$system.ram} 
            temp={$system.temp}
            disk={$system.disk}
            cpu={$system.cpu}
            gsVersion={$system.gsVersion}
            updateMode={$system.updateMode}
            />
        </div>

        <div class="panel" in:scale={{duration:120, delay: 200}}>
          <Network ethOnly={$system.ethOnly} connected={$system.connected} />
          <MinIO minio={$system.minio} />
          <Sessions sessions={$system.sessions} />
        </div>
      </div>

      <div class="main-panel {$isPortrait ? "portrait" : "landscape"}">
        <div class="panel" in:scale={{duration:120, delay: 200}}>
          <Power />
        </div>
        <div class="panel" in:scale={{duration:120, delay: 200}}>
          <Contact />
        </div>

      </div>
    {/if}

    {#if activeTab == 'Logs'}
      <div in:scale={{duration:120, delay: 200}}>
        <Logs container={selectedContainer} maxHeight="60vh" />
      </div>
      <div class="bottom-panel">
        <Listbox value={selectedContainer} on:change={(e) => (selectedContainer = e.detail)}>
          <ListboxOptions as="div" class="containers-list">
            {#each $system.containers as c}
              <ListboxOption as="p" value={c}>
                {c}
              </ListboxOption>
            {/each}
          </ListboxOptions>
          <ListboxButton class="containers-selector">{selectedContainer}</ListboxButton>
        </Listbox>
        <PrimaryButton on:click={exportLogs} standard="Export" status="standard" />
      </div>
    {/if}

  </Card>
{/if}

<style>
  .bottom-panel {
    padding-top: 24px;
    display: flex;
    align-items: end;
    gap: 12px;
  }
  :global(.containers-selector) {
    background: #FFFFFF4D;
    color: white;
    padding: 8px;
    width: 360px;
    border-radius: 6px;
    font-size: 12px;
    position: relative;
  }
  :global(.containers-list) {
    position: absolute;
    bottom: 48px;
    font-size: 12px;
    background: #040404;
    color: white;
    padding: 6px 12px 6px 12px;
    width: calc(360px - 24px);
    border-radius: 6px;
  }

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
