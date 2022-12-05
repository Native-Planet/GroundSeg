<script>
  import Fa from 'svelte-fa'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import { createEventDispatcher } from 'svelte'
  import { scale } from 'svelte/transition'
  import { api } from '$lib/api'
  import PierDeletionCheck from '$lib/PierDeletionCheck.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, isPierDeletion, hasBucket, autostart, loomSize

  let exportButtonText = 'Export Urbit Pier',
    deleteButtonText = 'Delete Urbit Pier',
    isLoading = false, showInfo = false, showLoom = false,
    modLoomStatus = "standard", curLoomSize = loomSize

  const dispatch = createEventDispatcher()

  const exportUrbitPier = () => {
    exportButtonText = 'Compressing your pier'
    isLoading = true

		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
      credentials: "include",
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'export'}),
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
  const toggleLoom = () => showLoom = !showLoom

  const modLoomSize = () => {
    modLoomStatus = "loading"
		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
      credentials: "include",
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'loom','size':curLoomSize}),
    })
      .then(r => r.json())
      .then(d => {
        if (d == 200) {
          modLoomStatus = "success"
        } else {
          modLoomStatus = "failure"
        }
        setTimeout(()=> modLoomStatus = "standard", 3000)
      })
  }


  const toggleAutostart = () => {
    autostart = !autostart
		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
      credentials: "include",
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

  <div class="option-title">
    Loom Size
    <button class="question-mark" on:click={toggleLoom} >
      <Fa icon={faCircleQuestion} size="1.2x" />
    </button>
  </div>
  {#if showLoom}
    <div class="loom-info">
      Loom settings set the amount of memory your ship is allocated in megabytes. 
      Do not go below 2048MB if you do not know what you are doing!
    </div>
  {/if}

  <div class="loom">
    <input type="range" min="28" max="32" step="1" class="range" bind:value={curLoomSize}>
    <div class="labels">
      <div class="label">256</div>
      <div class="label">512</div>
      <div class="label">1024</div>
      <div class="label">2048</div>
      <div class="label">4096</div>
    </div>

    <PrimaryButton
      on:click={modLoomSize}
      noMargin={true}
      status={(loomSize == curLoomSize) && (modLoomStatus == "standard") ? "disabled" : modLoomStatus}
      standard="Modify Loom Size"
      success="Loom size modified!"
      failure="Something went wrong, try again"
      loading="Modifying..."
    />
  </div>

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

  .loom {
    display: flex;
    align-items: center;
    flex-direction: column;
    gap: 4px;
    margin-bottom: 12px;
  }

  .loom-info {
    font-size: 11px;
  }

  .range {
    -webkit-appearance: none;
    width: 180px;
    height: 11px;
    background: #ffffff4d;
    border-radius: 16px;
  }

  .range::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: calc(180px / 5);
    height: 16px;
    background: var(--action-color);
    cursor: pointer;
    border-radius: 16px;
  }

  .range::-moz-range-thumb {
    width: calc(180px / 5);
    height: 16px;
    background: var(--action-color);
    cursor: pointer;
    border-radius: 16px;
  }

  .labels {
    display: flex;
    justify-content: space-between;
    width: 180px;
    padding-bottom: 6px;
  }

  .label {
    width: calc(180px / 5);
    font-size: 11px;
  }

</style>
