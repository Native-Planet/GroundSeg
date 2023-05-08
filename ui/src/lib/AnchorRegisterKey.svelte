<script>
  import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'
  import { send, socket, socketInfo } from '$lib/stores/websocket.js'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  // legacy
  let view = false
  let loading = false
  let buttonStatus = 'standard'
  let reRegCheck = true

  // Connection status
  $: connected = ($socketInfo?.metadata?.connected) || false

  // Startram form
  $: form = ($socketInfo?.forms?.startram) || null

  // Services
  $: urbits = ($socketInfo?.urbits) || {}
  /*
  const getRegistered = v => {
    if (
      (v.urbit_web == "unregistered") &&
      (v.urbit_ames == "unregistered") &&
      (v.minio == "unregistered")
    ) { return "unregistered" } else {
      return "registered
    }
  }
  */

  // Startram information
  $: startram = ($socketInfo?.system?.startram) || null
  $: register = (startram?.register) || "no"

  // Registration Key Logic
  $: key = updateForm('key',handleKey(key))
  const handleKey = key => {
    if (typeof key === 'string' || key instanceof String) {
      return key.trim()
    } else {return ''}
  }

  // Region Logic
  $: region = (form?.region) || null
  $: regions = (startram?.regions) || []

  // Send to API
  const updateForm = (item, data) => {
    if (connected) {
      let payload = {
        "category": "forms",
        "payload": {
          "template": "startram",
          "item": item,
          "value": data,
        }
      }
      send($socket, $socketInfo, document.cookie, payload)
    }
    return data
  }

  const registerKey = () => {
    let payload = {
      "category": "system",
      "payload": {
        "module": "startram",
        "action": "register"
      }
    }
    send($socket, $socketInfo, document.cookie, payload)
  }

  // Registration Key input visibility
  const toggleView = () => {
    view = !view
    document.querySelector('#input').type = view ? 'text' : 'password'
  }

  // Load up saved form
  onMount(()=> init())
  const init = () => form == null 
    ? setTimeout(init,100)
    : key = form.key

</script>

<div class="reg-key-wrapper">

  <!-- If not registered -->
  {#if register == "no"}
    <div class="reg-title" transition:scale={{duration:120, delay: 200}}>StarTram Key Registration</div>
    <div class="reg-key" transition:scale={{duration:120, delay: 200}}>
      <input id='input' placeholder="NativePlanet-some-word-another-word" type="password" bind:value={key} />
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>

    {#if regions.length > 0}
      <div class="reg-title" transition:scale={{duration:120, delay: 200}}>Region</div>
      <div class="regions-wrapper">
        {#each regions as r}
          <div 
            on:click={()=>updateForm("region",r.name)}
            class="region"
            class:region-active={region == null ? r.name == "us-east" : r.name == region}
            >
            {r.desc}
          </div>
        {/each}
      </div>
    {/if}

  <!-- if registered -->
  {:else if !reRegCheck}
    <div class="reg-title" transition:scale={{duration:120, delay: 200}}>StarTram Key Registration</div>
    <div class="reg-key" transition:scale={{duration:120, delay: 200}}>
      <input id='input' placeholder="NativePlanet-some-word-another-word" type="password" bind:value={key} />
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>

    {#if regions != null}
      <div class="reg-title" transition:scale={{duration:120, delay: 200}}>Region</div>
      <div class="regions-wrapper">
        {#each regions as r}
          <div 
            on:click={()=>updateForm("region",r.name)}
            class="region"
            class:region-active={region == null ? r.name == "us-east" : r.name == region}
            >
            {r.desc}
          </div>
        {/each}
      </div>
    {/if}
  {/if}

  <div class="ship-table">
    <div class="row-title">
      <div class="col-0 service-title">Urbit ID</div>
      <div class="col-1 service-title">Set to Remote</div>
    </div>
    {#each Object.entries(urbits) as [k,v]}
      <div class="row">
        <div class="col-0 heading">{k}</div>
        <div class="col-1">checkbox</div>
      </div>
    {/each}
    <div class="row">
      <div class="col-0 heading">All ships</div>
      <div class="col-1">checkbox</div>
    </div>
  </div>

  <!-- Submit button -->
  <div transition:scale={{duration:120, delay: 200}}>
    {#if (register == "yes") && reRegCheck}
      <PrimaryButton
        left={true}
        on:click={()=>reRegCheck = false}
        standard="Register Another Key or Change Region"
        status="standard"
        top="16"
      />
    {:else}
      <PrimaryButton
        left={true}
        standard="Register"
        status={key.length <= 0 ? "disabled" : 'standard'}
        top="12"
        on:click={registerKey}
      />
    {/if}
  </div>
</div>

<style>
  .reg-key-wrapper {
    gap: 6px;
    margin-top: 12px;
  }
  .reg-title {
    font-size: 14px;
    margin-bottom: 16px;
  }
  .reg-key {
    display: flex;
    margin-bottom: 18px;
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
  input::placeholder {
    color: white;
    opacity: .6;
  }
  .reg-key > img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }
  .regions-wrapper {
    display: flex;
    margin: 12px 0 30px 0;
    gap: 12px;
    border-radius: 4px;
  }
  .region {
    flex: 1;
    font-size: 12px;
    text-align: center;
    padding: 8px;
    border: solid 1px white;
    border-radius: 4px;
    cursor: pointer;
  }
  .region-active {
    background: #008eff;
  }
  .ship-table {
    display: flex;
    flex-direction: column;
  }
  .row-title {
    display: flex;
    font-size: 14px;
    line-height: 24px;
    text-align: center;
    gap: 4px;
  }
  .row {
    display: flex;
    font-size: 13px;
    line-height: 24px;
    text-align: center;
  }
  .row:hover {background: #0000004D;}
  .col-0 {flex: 4;text-align:right;}
  .col-1 {flex: 5;}
  .heading {
    padding: 4px 0;
  }
</style>
