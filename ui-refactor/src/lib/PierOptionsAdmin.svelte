<script>
  import { createEventDispatcher } from 'svelte'
  import { scale } from 'svelte/transition'
  import { api } from '$lib/api'
  import PierDeletionCheck from '$lib/PierDeletionCheck.svelte'

  export let name, isPierDeletion, hasBucket

  let exportButtonText = 'Export Urbit Pier',
    deleteButtonText = 'Delete Urbit Pier',
    isLoading = false

  const dispatch = createEventDispatcher()

  const exportUrbitPier = () => {
    exportButtonText = 'Compressing your pier...'
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

  const deleteUrbitPier = () => dispatch('delete') 

</script> 

{#if isPierDeletion}
  <div in:scale={{duration:120, delay: 600}}>
    <PierDeletionCheck {name} {hasBucket} on:cancel={()=>dispatch('delete')} /> 
  </div>
{:else}
  <div class="option-title">Admin Actions</div>
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
</style>
