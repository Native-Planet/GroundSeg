<script>
  import { api } from '$lib/api'
  import { scale } from 'svelte/transition'
  import { createEventDispatcher } from 'svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  let key = '',
    keyView = false,
    epKey = '',
    defaultEpKey = 'api.startram.io',
    buttonStatus = 'standard'

  epKey = defaultEpKey

  const dispatch = createEventDispatcher()

  const insertNP = () => epKey = defaultEpKey

  const toggleKeyView = () => {
    keyView = !keyView
    document.querySelector('#key-input').type = keyView ? 'text' : 'password'
  }

  const submitAnchor = () => {
    let step = "anchor"
    let query = {"key":key, "endpoint":epKey}

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
          setTimeout(()=> dispatch("nextPage"), 3000)
        }
        setTimeout(()=> buttonStatus = "standard", 3000)
      })
  }

</script>

<div class="title" in:scale={{duration:120, delay: 200}}>StarTram Key Registration</div>
<div class="reg-key" in:scale={{duration:120, delay: 200}}>
  <input id='key-input' type="password" bind:value={key} />
  <img on:click={toggleKeyView} src="/eye-{keyView ? "closed" : "open"}.svg" alt="eye" />
</div>

<div class="title">Startram Endpoint</div>
<div class="ep-key" in:scale={{duration:120, delay: 200}}>
  <input type="text" bind:value={epKey} />
  <img on:click={insertNP} width="24px" src="/nplogo.svg" alt="np logo" />
</div>

<div class="button">
  <PrimaryButton
    background="#ffffff4d"
    standard="Skip"
    on:click={()=> dispatch('nextPage')}
  />

  {#if ((epKey.length > 0) && (key.length > 0))}
    <PrimaryButton
      left={false}
      status={buttonStatus}
      standard="Register"
      failure="Registration failed"
      success="Registered!"
      loading="Registering your key.."
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
</style>
