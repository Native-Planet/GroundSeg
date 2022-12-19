<script>
  import { api } from '$lib/api'
  import { scale } from 'svelte/transition'
  import { createEventDispatcher } from 'svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  let password = '',
    confirmPassword = '',
    passView = false,
    confirmView = false,
    buttonStatus = 'standard'

  const dispatch = createEventDispatcher()

  const togglePassView = () => {
    passView = !passView
    document.querySelector('#pass-input-0').type = passView ? 'text' : 'password'
  }

  const toggleConfirmView = () => {
    confirmView = !confirmView
    document.querySelector('#pass-input-1').type = confirmView ? 'text' : 'password'
  }

  const createPassword = () => {
    let step = "password"
    let query = {"password":confirmPassword}

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
          buttonStatus = 'success'
          setTimeout(()=> window.location.href = "/login", 3000)
        }
        setTimeout(()=> buttonStatus = "standard", 3000)
      })
  }

</script>

<div class="title" in:scale={{duration:120, delay: 200}}>Create New Password</div>

<div class="pass-key" in:scale={{duration:120, delay: 200}}>
  <input id='pass-input-0' type="password" bind:value={password} />
  <img on:click={togglePassView} src="/eye-{passView ? "closed" : "open"}.svg" alt="eye" />
</div>

{#if password.length > 0}
  <div class="pass-key" in:scale={{duration:120, delay: 200}}>
    <input placeholder="Confirm Password" id='pass-input-1' type="password" bind:value={confirmPassword} />
    <img on:click={toggleConfirmView} src="/eye-{confirmView ? "closed" : "open"}.svg" alt="eye" />
  </div>
{/if}

<div class="button">
  <PrimaryButton
    background="#ffffff4d"
    standard="Back"
    on:click={()=> dispatch('prevPage')}
  />

  {#if ((confirmPassword.length > 0) && (password == confirmPassword))}
    <PrimaryButton
      left={false}
      status={buttonStatus}
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
    padding: 12px;
  }
  .pass-key {
    display: flex;
    padding: 0 20px 20px 20px;
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
