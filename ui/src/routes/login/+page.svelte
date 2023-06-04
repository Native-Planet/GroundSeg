<script>
  import { login, structure, loginStatus } from '$lib/stores/websocket'
  import { scale } from 'svelte/transition'
  import { onMount, onDestroy } from 'svelte'
  import { api, updateState } from '$lib/api'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faLock } from '@fortawesome/free-solid-svg-icons'

	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import EyeButton from '$lib/EyeButton.svelte'

  let inView = false
  let loginPassword = ''
  let buttonStatus = 'standard'

  $: loginModule = ($structure.system?.login) || null
  $: remainder = (loginModule?.cooldown) || 0
  $: unlocked = (remainder <= 0)

  $: hours = Math.floor(remainder / 3600)
  $: minutes = Math.floor((remainder % 3600) / 60)
  $: seconds = Math.floor(remainder % 60)

  onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    inView = true
  })

	onDestroy(()=> inView = false)

  const handleLogin = async () => {
    login(loginPassword)
  }

</script>

<svelte:head><script src="/jsencrypt.min.js"></script></svelte:head>

{#if inView}
  <Card width="640px" bgColor={unlocked ? "" : "#3F00008D"}>
    <div class="main-wrapper">
      <div class="opened-wrapper" in:scale={{delay:400, duration: 120}}>
        <img src="/npfull.svg" alt="Native Planet Logo" />
        {#if unlocked}
          <div class="login-wrapper">
            <input
              id="login-password"
              bind:value={loginPassword}
              class="login-password"
              type="password"
              placeholder='Password'
              disabled={!unlocked}
              on:keydown={e => {
                if (e.key === 'Enter') {
                  handleLogin()
                }
            }}>
          </div>
          {#if $loginStatus == "success"}
            <div class="button-info" style="color:lime;">Success!</div>
          {:else if $loginStatus == "loading"}
            <div class="button-info">Attempting to login..</div>
          {:else if $loginStatus == "AUTH_FAILED"}
            <div class="button-info" style="color:red;">Incorrect credentials</div>
          {:else}
            <PrimaryButton
              top=24
              left={false}
              standard="Login"
              status={(loginPassword.length > 0) && unlocked ? 'standard' : 'disabled'}
              on:click={handleLogin}
            />
          {/if}
        {:else}
          <div class="locked">
            <div class="locked-icon"><Fa icon={faLock} size="4x" /></div>
            <div class="locked-text">{hours > 0 ? hours + " HOURS" : ""} {minutes > 0 ? minutes + " MINUTES" : ""} {seconds} SECONDS</div>
          </div>
        {/if}
      </div>
    </div>
  </Card>
{/if}

<style>
  .main-wrapper {
    text-align: center;
    margin: 40px 80px;
  }
  .login-wrapper {
    margin-top: 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
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
  input {
    text-align: center;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: 0.4;
  }
  input::placeholder {
    color: inherit;
    opacity: .6;
  }
  .locked {
    text-align: center;
    margin-top: 30px;
    line-height: 40px;
  }
  .locked-icon {
    color: red;
  }
  .locked-text {
    margin-top: 30px;
    font-size: 24px;
  }
  .button-info {
    line-height: 30px;
    font-size: 12px;
    margin-top: 24px;
  }
</style>
