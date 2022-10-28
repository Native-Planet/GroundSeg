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
    <ListboxButton on:click={getNetworks} >{selectedConnection.length > 0 ? selectedConnection : "Select a wifi network"}</ListboxButton>
    <ListboxOptions>
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
