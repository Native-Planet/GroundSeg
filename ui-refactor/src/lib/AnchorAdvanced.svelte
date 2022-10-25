<script>
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  let advanced = false,
    currentEpKey = '',
    epKey = '',
    defaultEpKey = 'api.startram.io',
    epButtonStatus = 'standard'


  const insertNP = () => epKey = defaultEpKey

  const toggleAdvanced = () => {
    if (!advanced) {getEndpoint()}
    advanced = !advanced
  }
  
  const getEndpoint = () => {
    let module = 'anchor'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'get-url'})
	  })
      .then(d=>d.json())
      .then(r=>{currentEpKey = r; epKey = r})
  }

  const connectEndpoint = () => {
    epButtonStatus = 'loading'
    let module = 'anchor'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'action':'change-url','url':epKey})
	  })
      .then(d=>d.json()).then(r=>{
        if (r === 200) {
          epButtonStatus = 'success'
          setTimeout(()=>{
            epButtonStatus = 'standard'
            getEndpoint()
          }, 3000)}
        if (r === 400) {
          epButtonStatus = 'failure'
          setTimeout(()=>epButtonStatus = 'standard', 3000)
   }})}

</script>
<div class="reg-key-wrapper">
  <div class="advanced" on:click={toggleAdvanced}>
    Advanced Options
    <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
  </div>

  {#if advanced}
    <div class="ep-title">Set Endpoint</div>
    <div class="ep-key">
      <input type="text" bind:value={epKey} />
      <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
    </div>

    {#if currentEpKey != epKey}
      <PrimaryButton
        on:click={connectEndpoint}
        standard="Set to {defaultEpKey == epKey ? "Native Planet" : "Custom"} Endpoint"
        success="Endpoint successfully changed"
        failure="Failed to change endpoint"
        loading="Changing to your new endpoint.."
        status={epButtonStatus}
        top="12"
      />
    {/if}
  {/if}
</div>

<style>
  .reg-key-wrapper {
    gap: 6px;
    margin-top: 12px;
  }
  .advanced {
    font-size: 14px;
    padding-top: 6px;
    cursor: pointer;
    width: 150px;
  }
  .advanced:hover {
    opacity: .6;
  }

  .ep-title {
    margin-top: 18px;
    font-size: 14px;
    padding-bottom: 6px;
  }
  .ep-key {
    display: flex;
  }
  .ep-key > input {
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
  .ep-key > img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }


</style>
