<script>
  import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'
  import { structure, updateForm, starTramEndpoint, starTramRestart, starTramCancel } from '$lib/stores/websocket'
  import Fa from 'svelte-fa'
  import { faTriangleExclamation, faChevronDown, faChevronUp } from '@fortawesome/free-solid-svg-icons'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  let advanced = false
  let defaultEpKey = 'api.startram.io'
  let epButtonStatus = 'standard'
  let cancelButtonStatus = 'standard'
  let restartButtonStatus = 'standard'
  let confirmCancel = false
  let view = false
  let restarting = false
  let showEpInfo = false

  $: connected = ($structure?.metadata?.connected) || false
  $: startram = ($structure.system?.startram) || null
  $: register = (startram?.register) || "no"
  $: container = (startram?.container) || "stopped"
  $: restart = (startram?.restart) || ""
  $: cancel = (startram?.cancel) || ""
  $: endpoint = (startram?.endpoint) || null

  $: form = ($structure?.forms?.startram) || null
  $: formEndpoint = (form?.endpoint) || ""
  $: formCancel = (form?.cancel) || ""

  $: currentEpKey = endpoint
  $: currentEpKey = modifyEpKey(currentEpKey)
  const modifyEpKey = e => {
    if (!endpointData.hasOwnProperty(e)) {
      updateStarTramForm("endpoint",e)
      return e
    }
  }

  const insertNP = () => currentEpKey = defaultEpKey
  const toggleAdvanced = () => advanced = !advanced

  // Cancel Key Logic
  $: regKey = updateStarTramForm('cancel',handleKey(regKey))
  const handleKey = key => {
    if (typeof key === 'string' || key instanceof String) {
      return key.trim()
    } else {
      return ''
    }
  }
  const cancelSubscription = () => {
    updateStarTramForm("cancel", regKey.trim())
    confirmCancel
      ? starTramCancel()
      : confirmCancel = !confirmCancel
  }

  const doRestart = () => {
    starTramRestart()
  }

  const connectEndpoint = () => {
    updateForm("endpoint", currentEpKey)
    starTramEndpoint()
  }

  const updateStarTramForm = (item, data) => {
    updateForm('startram',item,data)
    return data
  }

  const restartData = {
    "initializing":"Attempting to restart StarTram",
    "stopping":"StarTram is stopping",
    "starting":"StarTram is starting",
    "success":"Restart Succeeded!"
  }

  const endpointData = {
    "stopping":"Stopping StarTram",
    "rm-services":"Removing Existing Services",
    "reset-pubkey":"Resetting Public Key",
    "changing":"Modifying Endpoint URL",
    "updating":"Updating Local Data",
    "success":"Endpoint Changed!"
  }

  const cancelData = {
    "cancelling":"Cancelling your subscription",
    "success":"Subscription Cancelled!",
    "failed":"An error has occured. Please send us a report"
  }

  const toggleView = () => {
    view = !view
    document.querySelector('#input-cancel').type = view ? 'text' : 'password'
  }

  // Load up saved form
  onMount(()=> init())
  const init = () => form == null 
    ? setTimeout(init,100)
    : regKey = formCancel

</script>

<div class="reg-key-wrapper">
  <div class="advanced" on:click={toggleAdvanced} transition:scale={{duration:120, delay: 200}}>
    Advanced Options
    <Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
  </div>

  {#if advanced}
    {#if container == "running"}
      <div class="ep-title" transition:scale={{duration:120, delay: 200}}>Restart StarTram</div>
      <div transition:scale={{duration:120, delay: 200}}>
        {#if restart === ""}
          <PrimaryButton
            on:click={doRestart}
            background="black"
            standard="Restart"
            top="12"
          />
        {:else}
          <div class="restart {restart}">{(restartData?.[restart]) || "error: " + restart}</div>
        {/if}
      </div>
    {/if}

    <div class="ep-title" transition:scale={{duration:120, delay: 200}}>
      Set Endpoint
      {#if register == "yes"}
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

    {#if !endpointData.hasOwnProperty(endpoint)}
      <div class="ep-key">
        <input type="text" bind:value={currentEpKey} />
        <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
      </div>

      <div>
        <PrimaryButton
          on:click={connectEndpoint}
          standard="Set to {defaultEpKey == currentEpKey ? "Native Planet" : "Custom"} Endpoint"
          status={currentEpKey == endpoint ? 'disabled': 'standard'}
          top="12"
        />
      </div>
    {:else}
      <div class="ep-key blocked">
        <input type="text" bind:value={formEndpoint} />
        <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
      </div>
      <div class="buttons {endpoint}">{(endpointData?.[endpoint]) || "error: " + endpoint}</div>
    {/if}

    {#if register == "yes"}
      <div class="ep-title" transition:scale={{duration:120, delay: 200}}>
        Cancelation
      </div>

      <div class="reg-key" transition:scale={{duration:120, delay: 200}}>
        <input id='input-cancel' placeholder="NativePlanet-some-word-another-word" type="password" bind:value={regKey} />
        <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
      </div>

      <div transition:scale={{duration:120, delay: 200}}>
        {#if !cancelData.hasOwnProperty(cancel)}
          <PrimaryButton
            on:click={cancelSubscription}
            background="#bb3f3f"
            standard="
              {
                confirmCancel ? "Click again to cancel your" : "Cancel my"
              } {
                defaultEpKey == currentEpKey ? "StarTram" : "Anchor"
              } subscription"
            status={regKey == "" ? 'disabled' : 'standard'}
            top="12"
          />
        {:else}
          <div class="buttons {cancel}">{(cancelData?.[cancel]) || "error: " + cancel}</div>
        {/if}
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
  input::placeholder {
    color: white;
    opacity: .6;
  }
  .restart {
    font-size: 12px;
    line-height: 24px;
    padding-top: 12px;
    animation: breathe 2s infinite;
  }
  .buttons {
    font-size: 12px;
    line-height: 18px;
    padding: 12px 0 6px 0;
    animation: breathe 2s infinite;
  }
  .success {
    color: lime;
    animation: none;
  }
  .blocked {
    pointer-events: none;
  }
</style>
