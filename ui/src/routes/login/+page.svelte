<script>
  import { scale } from 'svelte/transition'
  import { onMount, onDestroy } from 'svelte'
  import { api, updateState } from '$lib/api'

	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import EyeButton from '$lib/EyeButton.svelte'

  export let data
  updateState(data)

  let inView = false,
    showLogin = false,
    pwdView = false,
    loginPassword = ''


	onMount(()=> inView = true)
	onDestroy(()=> inView = false)

  const openLogin = () => showLogin = !showLogin

  const togglePwdView = () => {
    pwdView = !pwdView
    document.querySelector('#login-password').type = pwdView ? 'text' : 'password'
  }

  const handleLogin = () => {
    fetch($api + '/login', {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'password':loginPassword})
	  })
      .then(r => r.json())
      .then(d => { 
        console.log(d)
        /*
        if (d == 200) {buttonStatus = 'success'}
        else {buttonStatus = 'failure'}
        setTimeout(()=> {
          buttonStatus = 'standard', 10000
          showButton = false
        })
        */
    })}

</script>

{#if inView}
<Card width="640px">
  <div class="main-wrapper">

    {#if !showLogin}

      <div class="opened-wrapper" in:scale={{delay:400, duration: 120}}>

        <div class="login-wrapper">
          <input
            id="login-password"
            bind:value={loginPassword}
            class="login-password"
            type="password"
          />
        </div>

        <div class="buttons">
          <PrimaryButton
            top=24
            left={false}
            standard="Submit Password"
            on:click={handleLogin}
          />
          <PrimaryButton
            top=12
            standard="Cancel"
            background="none"
            on:click={openLogin}
          />
        </div>

      </div>

    {:else}
      <div class="closed-wrapper" in:scale={{delay:400, duration: 120}}>
        <img src="/npfull.svg" alt="Native Planet Logo" />
        <div>
          <PrimaryButton
            standard="Login"
            on:click={openLogin}
          />
        </div>
      </div>
    {/if}
  </div>
</Card>
{/if}

<style>
  .main-wrapper {
    text-align: center;
    margin: 120px;
  }
  .closed-wrapper {
    display: flex;
    flex-direction: column;
    gap: 24px;
    align-items: center;
  }
  .login-wrapper {
    display: flex;
  }
  .login-password {
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
    border: none;
    font-family: inherit;
    color: inherit;
  }
  input:focus {
    outline: none;
  }
</style>
