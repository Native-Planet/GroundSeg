<script>
  import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'
  import { connected, structure, updateForm, starTramRegister } from '$lib/stores/websocket'

  import CheckBox from '$lib/CheckBox.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  // legacy
  let view = false
  let loading = false
  let buttonStatus = 'standard'
  let reRegCheck = true

  // Startram form
  $: form = ($structure?.forms?.startram) || null

  // Ships
  $: urbits = ($structure?.urbits) || {}
  $: unchecked = (form?.ships) || []

  // Startram information
  $: startram = ($structure?.system?.startram) || null
  $: register = (startram?.register) || "no"

  // Registration Key Logic
  $: key = updateStarTramForm('key',handleKey(key))
  const handleKey = key => {
    if (typeof key === 'string' || key instanceof String) {
      return key.trim()
    } else {return ''}
  }

  // Region Logic
  $: region = (form?.region) || null
  $: regions = (startram?.regions) || []

  // Send to API
  const updateStarTramForm = (item, data) => {
    updateForm('startram',item,data)
    return data
  }

  // Toggle remote access after registration
  const addShip = e => {
    updateStarTramForm("ships",[e.detail])
  }
  const addAllShips = e => {
    updateStarTramForm("ships", e.detail ? "none" : "all")
  }

  const registerKey = () => {
    // final send just in case
    updateStarTramForm("key",key)
    starTramRegister()
  }

  // Registration Key input visibility
  const toggleView = () => {
    view = !view
  }

  // Load up saved form
  onMount(()=> init())
  const init = () => form == null 
    ? setTimeout(init,100)
    : key = form.key

  const options = {
    "registering":"Registering your StarTram key",
    "updating":"Retrieving updated information from StarTram API",
    "start-wg":"Starting Wireguard connecion",
    "start-mc":"Initializing MinIO",
    "success":"Successfully Registered Your StarTram Key"
  }

</script>

<div class="reg-key-wrapper">
  <!-- If not registered -->
  {#if register == "no"}
    <div class="reg-title" transition:scale={{duration:120, delay: 200}}>StarTram Key Registration</div>
    <div class="reg-key" transition:scale={{duration:120, delay: 200}}>
      {#if view}
        <input id='input' placeholder="NativePlanet-some-word-another-word" type="text" bind:value={key} />
      {:else}
        <input id='input' placeholder="NativePlanet-some-word-another-word" type="password" bind:value={key} />
      {/if}
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>

    {#if regions.length > 0}
      <div class="reg-title" transition:scale={{duration:120, delay: 200}}>Select a Region</div>
      <div class="regions-wrapper">
        {#each regions as r}
          <div 
            on:click={()=>updateStarTramForm("region",r.name)}
            class="region"
            class:region-active={region == null ? r.name == "us-east" : r.name == region}
            >
            {r.desc}
          </div>
        {/each}
      </div>
    {/if}

    {#if Object.entries(urbits).length > 0} 
      <div class="reg-title" transition:scale={{duration:120, delay: 200}}>Automatically Set to Remote</div>
      <div class="ship-table">
        {#each Object.entries(urbits) as [k,v]}
          <CheckBox name={k} check={!unchecked.includes(k)} on:update={addShip} submitting={false} />
        {/each}
        {#if Object.entries(urbits).length > 1} 
          <CheckBox all={true} check={unchecked.length <= 0} on:update={addAllShips} submitting={false} />
        {/if}
      </div>
    {/if}

  <!-- if registered -->
  {:else if (!reRegCheck) && (register == "yes")}
    <div class="reg-title" transition:scale={{duration:120, delay: 200}}>StarTram Key Registration</div>
    <div class="reg-key" transition:scale={{duration:120, delay: 200}}>
      {#if view}
        <input id='input' placeholder="NativePlanet-some-word-another-word" type="text" bind:value={key} />
      {:else}
        <input id='input' placeholder="NativePlanet-some-word-another-word" type="password" bind:value={key} />
      {/if}
      <img on:click={toggleView} src="/eye-{view ? "closed" : "open"}.svg" alt="eye" />
    </div>

    {#if regions != null}
      <div class="reg-title" transition:scale={{duration:120, delay: 200}}>Select a Region</div>
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

    <div class="reg-title" transition:scale={{duration:120, delay: 200}}>Automatically Set to Remote</div>
    <div class="ship-table">
      {#each Object.entries(urbits) as [k,v]}
        <CheckBox name={k} check={!unchecked.includes(k)} on:update={addShip} submitting={false} />
      {/each}
      {#if Object.entries(urbits).length > 1} 
        <CheckBox all={true} check={unchecked.length <= 0} on:update={addAllShips} submitting={false} />
      {/if}
    </div>

  {/if}

  <!-- Submit button -->
  <div transition:scale={{duration:120, delay: 200}}>
    {#if register == "yes"}
      <PrimaryButton
        left={true}
        on:click={()=> !reRegCheck ? registerKey() : reRegCheck = false}
        standard={reRegCheck ? "Register Another Key or Change Region" : "Register"}
        top="16"
      />
    {:else if register == "no"}
      <PrimaryButton
        left={true}
        standard="Register"
        status={key.length <= 0 ? "disabled" : 'standard'}
        top="12"
        on:click={registerKey}
      />
    {:else}
      {#if options.hasOwnProperty(register)}
        <div class="loading {register}">{options[register]}</div>
      {:else if register.includes("failure")}
        <div class="loading failure">{register.split("\n")[1]}</div>
      {/if}
      {#each Object.entries(urbits) as [k,v]}
        <div class="row">
          <div class="ship">{k}</div>
          {#if (v.startram.urbit == "registering") || (v.startram.minio == "registering")}
            <div class="ship-status">Registering Services</div>
          {:else if v.startram.access == "to-remote"}
            <div class="ship-status">Toggling Remote Access</div>
          {:else if v.startram.access == "to-local"}
            <div class="ship-status">Toggling Local Access</div>
          {:else if (v.startram.access == "remote")}
            <div class="ship-status remote">Remote Acess Active</div>
          {:else if (v.startram.access == "local")}
            <div class="ship-status remote">Local Access Active</div>
          {:else if (v.startram.urbit == "unregistered") || (v.startram.minio == "unregistered")}
            <div class="ship-status">Awaiting Registration</div>
          {:else if (v.startram.urbit == "registered") || (v.startram.minio == "registered")}
            <div class="ship-status">Services Registered</div>
          {:else}
            <div class="ship-status unknown">
              <div>Unknown Status</div>
              <div>Non-pretty version of data:</div>
              <div>{JSON.stringify(v.startram)}</div>
            </div>
          {/if}
        </div>
      {/each}
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
  .loading {
    animation: breathe 2s infinite;
    text-align: center;
    font-size: 14px;
    margin: 40px;
  }
  .success {
    color: lime;
    animation: none;
  }
  .failure {
    color: red;
    animation: none;
  }
  .row {
    display: flex;
    font-size: 12px;
    gap: 24px;
    padding: 8px;
  }
  .row:hover {
    background: #0000004D;
  }
  .ship {
    flex: 1;
    text-align: right;
  }
  .ship-status {
    flex: 1;
    text-align: left;
  }
  .unknown {
    color: orange;
  }
  .remote {
    color: lime;
  }
</style>
