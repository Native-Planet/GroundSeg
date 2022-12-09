<script>
  import { scale } from 'svelte/transition'
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faTriangleExclamation, faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let wgReg

  let advanced = false,
    currentEpKey = '',
    epKey = '',
    defaultEpKey = 'api.startram.io',
    epButtonStatus = 'standard',
    cancelButtonStatus = 'standard',
    confirmCancel = false,
    regKey = '', view = false,
    showEpInfo = false


  const insertNP = () => epKey = defaultEpKey

  const toggleAdvanced = () => {
    if (!advanced) {getEndpoint()}
    advanced = !advanced
  }
  
  const getEndpoint = () => {
    let module = 'anchor'
	  fetch($api + '/system?module=' + module, {
			method: 'POST',
      credentials: "include",
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
      credentials: "include",
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

  const cancelSubscription = () => {
    if (confirmCancel) {
      cancelButtonStatus = 'loading'
      let module = 'anchor'

  	  fetch($api + '/system?module=' + module, {
	  		method: 'POST',
        credentials: "include",
		  	headers: {'Content-Type': 'application/json'},
			  body: JSON.stringify({'action':'unsubscribe','key':regKey.trim()})
  	  })
       .then(d=>d.json())
        .then(r=>{
          if (r == 200) {
            cancelButtonStatus = 'success'
            regKey = ''
          } else {
            cancelButtonStatus = 'failure'
          }
          setTimeout(()=> cancelButtonStatus = 'standard', 3000)
          confirmCancel = !confirmCancel
       })
    } else {
      confirmCancel = !confirmCancel
  }}

  const toggleView = () => {
    view = !view
    document.querySelector('#input').type = view ? 'text' : 'password'
  }

</script>

<div class="reg-key-wrapper">
  <div class="advanced" on:click={toggleAdvanced} transition:scale={{duration:120, delay: 200}}>
    Advanced Options
    <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
  </div>

  {#if advanced}
    <div class="ep-title" transition:scale={{duration:120, delay: 200}}>
      Set Endpoint
      {#if wgReg}
        <button class="alert-mark" on:click={()=>showEpInfo = !showEpInfo} >
          <Fa icon={faTriangleExclamation} size="1.2x" />
        </button>
      {/if}
    </div>

    {#if showEpInfo}
      <div class="ep-info">
        Modifying your endpoint will result in removing all StarTram related services attached to this device.
      </div>
    {/if}

    <div class="ep-key"transition:scale={{duration:120, delay: 200}}>
      <input type="text" bind:value={epKey} />
      <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
    </div>

    {#if currentEpKey != epKey}
      <div transition:scale={{duration:120, delay: 200}}>
      <PrimaryButton
        on:click={connectEndpoint}
        standard="Set to {defaultEpKey == epKey ? "Native Planet" : "Custom"} Endpoint"
        success="Endpoint successfully changed"
        failure="Failed to change endpoint"
        loading="Changing to your new endpoint.."
        status={epButtonStatus}
        top="12"
      />
      </div>
    {/if}

    {#if wgReg}
    <div class="ep-title" transition:scale={{duration:120, delay: 200}}>
      Cancelation
    </div>

    <div class="reg-key" transition:scale={{duration:120, delay: 200}}>
      <input placeholder="Registration Key" id='input' type="password" bind:value={regKey} />
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>

    <div transition:scale={{duration:120, delay: 200}}>
      <PrimaryButton
        on:click={cancelSubscription}
        background="#bb3f3f"
        standard="
          {
            confirmCancel ? "Click again to cancel your" : "Cancel my"
          } {
            defaultEpKey == currentEpKey ? "StarTram" : "Anchor"
          } subscription"
        success="Subscription successfully canceled!"
        failure="Something went wrong, please try again"
        loading="Canceling your subscription"
        status={regKey.length < 1 ? 'disabled' : cancelButtonStatus}
        top="12"
      />
    </div>
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
  .reg-key {
    display: flex;
  }
  .reg-key > input {
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
  .reg-key > img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }
  .ep-info {
    font-size: 11px;
    padding-bottom: 12px;
    color: orange;
  }
  .alert-mark {
    cursor: pointer;
    color: orange;
  }
</style>
