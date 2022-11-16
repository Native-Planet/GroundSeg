<script>
  import { onMount, onDestroy } from 'svelte'
	import { api } from '$lib/api'
  import { Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@rgossiaux/svelte-headlessui"

  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import EyeButton from '$lib/EyeButton.svelte'

  export let connected

  let networks = [],
    opened = false,
    view = false,
    refreshing = false,
    buttonStatus = 'standard',
    selectedConnection = connected,
    nw = '', pw = ''

  const getNetworks = () => {
    if (opened) {
    let module = 'network'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'networks'})
	  })
      .then(r => r.json()).then(d => networks = d)
      setTimeout(getNetworks, 10000)
    }}

  const connectToNetwork = () =>  {

    let module = 'network'
    buttonStatus = 'loading'

	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: 'include',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'connect','network':selectedConnection,'password':pw})
	  })
      .then(r => r.json())
      .then(d => { if (d == 200) {
        buttonStatus = 'success'
        setTimeout(()=>{
          buttonStatus = 'standard'
          selectedConnection = connected
          pw = ''
        }, 3000)
      } else {
        buttonStatus = 'failure'
        setTimeout(()=>{
          selectedNetwork = connected
          buttonStatus = 'standard'
        }, 3000)
      }})}

  const toggleView = () => view = !view

  onMount(()=> opened = true)
  onDestroy(()=> opened = false)

</script>
<div class="wifi">
  <Listbox value={selectedConnection} on:change={(e) => (selectedConnection = e.detail)}>
    <ListboxButton class="connection-selector" on:click={getNetworks} >{selectedConnection.length > 0 ? selectedConnection : "Select a wifi network"}</ListboxButton>
    <ListboxOptions class="connection-list">
      {#each networks as network}
        <ListboxOption value={network}>
          {network}
        </ListboxOption>
      {/each}
    </ListboxOptions>
  </Listbox>

  {#if (connected !== selectedConnection) && (selectedConnection.length > 0)}

    <div class="wifi-pass-wrapper">
      <div class="pass-text">Wifi Password</div>
      <div class="wifi-pass">
        <input id='pass' type="password" bind:value={pw} />
 		    <EyeButton on:click={toggleView} {view} />
      </div>
      <PrimaryButton
        on:click={connectToNetwork}
        standard="Connect"
        success="Connected to network"
        failure="Connection failed"
        loading="Connecting"
        status={(pw == '') || (selectedConnection == null)? "disabled" : buttonStatus}
        top="12" />
    </div>
  {/if}
</div>
 
<style>
	:global(.connection-list::-webkit-scrollbar) {display: none;}
  :global(.connection-selector) {
    padding: 8px 0 8px 0;
    font-size: 12px;
    font-family: inherit;
    color: inherit;
    background: #FFFFFF4D;
    border-radius: 6px;
    position: relative;
    width: 100%;
  }
  :global(.connection-list) {
    font-size: 12px;
    line-height: 24px;
    margin: auto;
    width: 300px;
    text-align: center;
    border-radius: 6px;
    background: #040404;
    position: absolute;
    padding: 0 6px 0 6px;
    max-height: 240px;
    -ms-overflow-style: none;
		scrollbar-width: none;
    overflow: scroll;
    list-style-type: none;
  }

  .wifi {
    margin-top: 20px;
  }
  .wifi-pass-wrapper {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-top: 12px;
  }
  .pass-text {
    font-size: 12px;
  }
  .wifi-pass {
    display: flex;
  }
  .wifi-pass > input {
    font-family: inherit;
    background: #ffffff4d;
    color: inherit;
    border-radius: 6px;
    font-size: 12px;
    padding: 8px;
    border: none;
    flex: 1;
  }
  input:focus {
    outline: none;
  }
</style>
