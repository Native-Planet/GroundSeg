<script>
  import { page } from '$app/stores'
  import { onMount } from 'svelte'
  import { api } from '$lib/api'
  import { scale } from 'svelte/transition'
  import { createEventDispatcher } from 'svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  let password = ''
  let confirmPassword = ''
  let passView = false
  let confirmView = false
  let buttonStatus = 'standard'
  let pubKey = ''

  const dispatch = createEventDispatcher()

  const togglePassView = () => {
    passView = !passView
    document.querySelector('#pass-input-0').type = passView ? 'text' : 'password'
  }

  const toggleConfirmView = () => {
    confirmView = !confirmView
    document.querySelector('#pass-input-1').type = confirmView ? 'text' : 'password'
  }

  const createPassword = async () => {
    let step = "password"
    let query = await encryptPassword(pubKey)

    buttonStatus = "loading"

    fetch($api + '/setup?page=' + step, {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify(query)
    })
      .then(d => d.json())
      .then(res => {
        console.log(res)
        if (res == 400) {
          window.location.href = "/login"
        }

        if (res == 401) {
          buttonStatus = "failure"
        }

        if (res == 200) {
          buttonStatus = 'success'
          dispatch('nextPage')
        }
        setTimeout(()=> buttonStatus = "standard", 3000)
      })
  }

  const getLoginKey = () => {
    if ($page.url.pathname == "/setup") {
      fetch($api + '/login/key')
      .then(r => r.json())
        .then(d => {
          pubKey = d
        })
      setTimeout(getLoginKey, 30000)
  }}

  const encryptPassword = async pub => {
    const encrypt = new JSEncrypt({ default_key_size: 2048 })
    encrypt.setPublicKey(pub)
    const encrypted = await encrypt.encrypt(confirmPassword.trim())
    return {"password":encrypted,"pubkey":pub}
  }

  onMount(()=> getLoginKey())

</script>

<svelte:head><script src="/jsencrypt.min.js"></script></svelte:head>

<div class="title" in:scale={{duration:120, delay: 200}}>Create a password</div>

<div class="pass-key" in:scale={{duration:120, delay: 200}}>
  <input id='pass-input-0' placeholder="new password" type="password" bind:value={password} />
  <img on:click={togglePassView} src="/eye-{passView ? "closed" : "open"}.svg" alt="eye" />
</div>

<div class="pass-key" in:scale={{duration:120, delay: 200}}>
  <input placeholder="confirm your password" id='pass-input-1' type="password" bind:value={confirmPassword} />
  <img on:click={toggleConfirmView} src="/eye-{confirmView ? "closed" : "open"}.svg" alt="eye" />
</div>

<div class="button">
  {#if ((confirmPassword.length > 0) && (password == confirmPassword))}
    <PrimaryButton
      left={false}
      status={pubKey.length > 0 ? buttonStatus : 'disabled'}
      standard="Set Password"
      failure="Something went wrong"
      success="Password set!"
      loading="Setting your password.."
      on:click={createPassword}
    />
  {/if}
</div>

<style>
  .title {
    text-align: center;
    padding: 8px 0 18px 0;
  }
  .pass-key {
    display: flex;
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
    margin-bottom: 20px; 
  }
  input:focus {
    outline: none;
  }
  input::placeholder {
    color: #ffffff;
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
