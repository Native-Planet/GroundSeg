<script>
  import Fa from 'svelte-fa'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import { createEventDispatcher } from 'svelte'
  import { scale } from 'svelte/transition'
  import { api } from '$lib/api'
  import PierDeletionCheck from '$lib/PierDeletionCheck.svelte'

  export let name, isPierDeletion, hasBucket, autostart

  let exportButtonText = 'Export Urbit Pier',
    deleteButtonText = 'Delete Urbit Pier',
    isLoading = false, showInfo = false

  const dispatch = createEventDispatcher()

  const exportUrbitPier = () => {
    exportButtonText = 'Compressing your pier'
    isLoading = true

		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'export'})
		  })
      .then(res => { return res.blob(); })
      .then(d => {
        isLoading = false
        exportButtonText = 'Your pier has been exported!'
        var a = document.createElement("a")
        a.href = window.URL.createObjectURL(d)
        a.download = name
        a.click()
        setTimeout(()=> exportButtonText = 'Export Urbit Pier', 5000)
  })}

  const toggleInfo = () => showInfo = !showInfo

  const toggleAutostart = () => {
    autostart = !autostart
		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'toggle-autostart'})
		  })
      .then(res => res.json())
      .then(d => {
        console.log(d)
  })}

  const deleteUrbitPier = () => dispatch('delete') 

</script> 

{#if isPierDeletion}

  <div in:scale={{duration:120, delay: 300}}>
    <PierDeletionCheck {name} {hasBucket} on:cancel={()=>dispatch('delete')} /> 
  </div>

{:else}

  <div class="option-title">Admin Actions</div>

  <div class="autostart" >
    <input type="checkbox" bind:checked={autostart} on:click={toggleAutostart} />
    <div class="autostart-text" on:click={toggleAutostart} >Remember Urbit Ship Status</div>

    <!-- Info button -->
    <button class="question-mark" on:click={toggleInfo} >
      <Fa icon={faCircleQuestion} size="1.2x" />
    </button>
  </div>

  {#if showInfo}
  <div class="info-text">
    Enabling this will allow your ship to be booted automatically after restarting your device.
  </div>
  {/if}

  <button class="export-pier" class:loading={isLoading} on:click={exportUrbitPier}>
    {exportButtonText}
  </button>
  <button class="delete-pier" on:click={deleteUrbitPier}>{deleteButtonText}</button>
{/if}

<style>
  .option-title {
    font-size: 14px;
    color: inherit;
    margin-bottom: 4px;
  }

  .export-pier {
    color: orange;
    cursor: pointer;
  }

  .loading {
    color: white;
    animation: breathe 2s infinite;
  }

  .delete-pier {
    color: red;
    cursor: pointer;
  }
  .autostart {
    display: flex;
    font-size: 12px;
    gap: 6px;
    align-items: center;
    justify-content: center;
    text-align: center;
    cursor: pointer;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
  }
  .info-text {
    font-size: 11px;
  }
</style>
