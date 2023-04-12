<script>
  import { createEventDispatcher } from 'svelte'
  import { Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@rgossiaux/svelte-headlessui"
  import { api, system } from '$lib/api'

  export let name, containers

  let selectedContainer = name

  const dispatch = createEventDispatcher()

  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import Logs from '$lib/Logs.svelte'

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
          var element = document.createElement('a')
          element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(d))
          element.setAttribute('download', selectedContainer)
          element.style.display = 'none'
          document.body.appendChild(element)
          element.click()
          document.body.removeChild(element)
      })
  }

</script>
<Logs container={selectedContainer} maxHeight="50vh"/>
      <div class="bottom-panel">
        <Listbox value={selectedContainer} on:change={(e) => (selectedContainer = e.detail)}>
          <ListboxOptions as="div" class="containers-list">
            {#each containers as c}
              <ListboxOption as="p" value={c}>
                {c}
              </ListboxOption>
            {/each}
          </ListboxOptions>
          <ListboxButton class="containers-selector">{selectedContainer}</ListboxButton>
        </Listbox>
        <PrimaryButton on:click={exportLogs} background="#FFFFFF4D" standard="Export" status="standard" />
      </div>

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

</style>
