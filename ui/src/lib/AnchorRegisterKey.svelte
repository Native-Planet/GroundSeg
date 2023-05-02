<script>
  import { scale } from 'svelte/transition'
  import { send, socket, socketInfo } from '$lib/stores/websocket.js'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let register
  export let region
  export let regions

  let view = false
  let loading = false
  let buttonStatus = 'standard'
  let reRegCheck = true

  $: connected = ($socketInfo?.metadata?.connected) || false
  $: key = updateForm('key',handleKey(key))

  const handleKey = key => {
    if (typeof key === 'string' || key instanceof String) {
      return key.trim()
    }
    else {
      return ''
    }
  }

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

  // Region
  let selectedRegion
  if (region == null) {
    selectedRegion = "us-east"
  } else {
    selectedRegion = region
  }

  const toggleView = () => {
    view = !view
    document.querySelector('#input').type = view ? 'text' : 'password'
  }


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
            on:click={()=>selectedRegion = r.name}
            class="region"
            class:region-active={r.name == selectedRegion}
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
            on:click={()=>selectedRegion = r.name}
            class="region"
            class:region-active={r.name == selectedRegion}
            >
            {r.desc}
          </div>
        {/each}
      </div>
    {/if}
  {/if}

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
      />
      <!--
        on:click={registerKey}
        success="Key registered"
        failure="Registration failed"
        loading="Registering your key..(might take a while)"
      -->
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
</style>
