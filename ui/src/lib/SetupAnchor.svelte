<script>
  import { api } from '$lib/api'
  import { scale } from 'svelte/transition'
  import { createEventDispatcher, beforeUpdate, afterUpdate } from 'svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  let key = ''
  let keyView = false
  let epKey = ''
  let defaultEpKey = 'api.startram.io'
  let buttonStatus = 'standard'
  let regions = null
  let selectedRegion = "us-east"

  epKey = defaultEpKey

  const dispatch = createEventDispatcher()

  const insertNP = () => epKey = defaultEpKey

  const toggleKeyView = () => {
    keyView = !keyView
    document.querySelector('#key-input').type = keyView ? 'text' : 'password'
  }

  const skipAnchor = () => {
    let step = "anchor"
    let query = {"skip":true}

    fetch($api + '/setup?page=' + step, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(query)
    })
      .then(d => d.json())
      .then(res => {
        if (res == 200) {
          window.location.href = "/login"
        } else {
          console.log("failed")
        }
      })
  }

  const getRegions = () => {
    let content = "regions" 
    let query = {"endpoint":epKey.trim()}

    fetch($api + '/setup?page=' + content, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(query)
    }).then(d => d.json())
      .then(res => {
        if (res.error == 0) {
          regions = res.regions
        } else {
          regions = null
          console.log("Error non-zero: " + JSON.stringify(res))
        }
      })
      .catch(err => console.log("Failed to submit endpoint: " + JSON.stringify(err)))
  }

  const submitAnchor = () => {
    let step = "anchor"
    let query = {"key":key, "endpoint":epKey, "region":selectedRegion}

    buttonStatus = "loading"

    fetch($api + '/setup?page=' + step, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(query)
    })
      .then(d => d.json())
      .then(res => {
        if (res == 400) {
          window.location.href = "/login"
        }

        if (res == 401) {
          buttonStatus = "failure"
        }

        if (res == 200) {
          buttonStatus = "success"
          window.location.href = "/login"
        }
        setTimeout(()=> buttonStatus = "standard", 3000)
      })
  }

  let oldEpKey = ''
  afterUpdate(()=> {
    if (oldEpKey != epKey) {
      getRegions()
      oldEpKey = epKey
    }
  })

</script>

<div class="title" in:scale={{duration:120, delay: 200}}>StarTram Key Registration</div>
<div class="reg-key" in:scale={{duration:120, delay: 200}}>
  <input placeholder="NativePlanet-some-word-another-word" id='key-input' type="password" bind:value={key} />
  <img on:click={toggleKeyView} src="/eye-{keyView ? "closed" : "open"}.svg" alt="eye" />
</div>

<div class="title">Startram Endpoint</div>
<div class="ep-key" in:scale={{duration:120, delay: 200}}>
  <input type="text" bind:value={epKey} />
  <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
</div>

{#if regions != null}
  <div class="title" transition:scale={{duration:120, delay: 200}}>Region</div>
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

<div class="sign-up">
  <a href="https://www.nativeplanet.io/startram" target="_blank">
    Need a startram registration key? Get one here!
  </a>
</div>

<div class="button">
  <PrimaryButton
    background="#ffffff4d"
    standard="Skip"
    on:click={skipAnchor}
  />

  {#if ((epKey.length > 0) && (key.length > 0))}
    <PrimaryButton
      left={false}
      status={buttonStatus}
      standard="Register"
      failure="Registration failed"
      success="Registered!"
      loading="Registering your key..(might take a while)"
      on:click={submitAnchor}
    />
  {/if}
</div>

<style>
  .title {
    text-align: center;
    padding: 12px;
  }
  .reg-key {
    display: flex;
    padding: 0 20px 0 20px;
  }
  .ep-key {
    display: flex;
    padding: 0 20px 0 20px;
  }
  input {
    text-align: center;
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
    color: #FFFFFF8D; 
  }
  img {
    padding-left: 12px;
    opacity: .8;
    cursor: pointer;
  }
  .button {
    padding-top: 24px;
    padding-right: 20px;
    display: flex;
  }
  .regions-wrapper {
    display: flex;
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
  .sign-up {
    margin-top: 16px;
    margin-left: 20px;
  }
  a {
    color: inherit;
    font-size: 12px;
    text-decoration: underline;
    cursor: pointer;
  }
</style>
