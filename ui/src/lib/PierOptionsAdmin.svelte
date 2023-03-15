<script>
  import Fa from 'svelte-fa'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'
  import { createEventDispatcher } from 'svelte'
  import { scale } from 'svelte/transition'
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name
  export let autostart

  let exportButtonText = 'Export Urbit Pier'
  let deleteButtonText = 'Delete Urbit Pier'
  let isLoading = false
  let showInfo = false

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

</script> 

<div class="bg">
  <div class="option-title">Admin Actions</div>

  <div class="autostart-wrapper">
    <div class="autostart" on:click={toggleAutostart}>
      <div class="box" class:highlight={autostart}>
        {#if autostart}
          <Fa icon={faCheck} size="1x"/>
        {/if}
      </div>
      Remember Urbit Ship Status
    </div>
    <!-- Info button -->
    <button class="question-mark" on:click={toggleInfo} >
      <Fa icon={faCircleQuestion} size="1x" />
    </button>
  </div>


  {#if showInfo}
  <div class="info-text">
    Enabling this will allow your ship to be booted automatically after restarting your device.
  </div>
  {/if}

  <div class="danger-zone">
    <button class="export-pier" class:loading={isLoading} on:click={exportUrbitPier}>
      {exportButtonText}
    </button>
    <button class="delete-pier" on:click={()=>dispatch('delete')}>{deleteButtonText}</button>
  </div>
</div>

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .option-title {
    font-size: 14px;
    color: inherit;
    margin-bottom: 12px;
  }

  .export-pier {
    padding: 12px;
    padding-bottom: 0;
    color: orange;
    cursor: pointer;
  }
  .delete-pier {
    padding: 12px;
    padding-bottom: 0;
    color: red;
    cursor: pointer;
  }
  .loading {
    color: white;
    animation: breathe 2s infinite;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
    padding-top: 2px;
  }
  .info-text {
    font-size: 11px;
  }
  .danger-zone {
    display: flex;
    justify-content: center;
  }
  .autostart-wrapper {
    display: flex;
    justify-content: center;
    gap: 6px;
    padding: 12px 0px;
  }
  .autostart {
    display: flex;
    gap: 6px;
    align-items: center;
    justify-content: center;
    text-align: center;
    font-size: 11px;
    cursor: pointer;
    user-select: none;
  }
  .box {
    width: 14px;
    height: 14px;
    background: #ffffff4d;
    border-radius: 4px;
  }
  .highlight {
    background: #028AFB;
  }
</style>
