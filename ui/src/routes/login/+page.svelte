<script>
  //import { login, structure, loginStatus } from '$lib/stores/websocket'
  import { login, structure } from '$lib/stores/websocket'
  import { wide } from '$lib/stores/display'
  import { scale } from 'svelte/transition'
  import { onMount, onDestroy } from 'svelte'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faLock } from '@fortawesome/free-solid-svg-icons'

  let inView = false
  let loginPassword = ''
  let buttonStatus = 'standard'

  $: loginModule = ($structure?.login) || null
  $: remainder = (loginModule?.remainder) || 0
  $: unlocked = (remainder <= 0)

  $: hours = Math.floor(remainder / 3600)
  $: minutes = Math.floor((remainder % 3600) / 60)
  $: seconds = Math.floor(remainder % 60)

  onMount(()=> inView = true)
	onDestroy(()=> inView = false)

  const handleLogin = async () => {
    login(loginPassword)
  }

</script>

<!--svelte:head><script src="/jsencrypt.min.js"></script></svelte:head-->

{#if inView}
  <div class="container {$wide ? "wide" : "slim"}">

    <!-- Unlocked -->
    {#if unlocked}
      <!-- Title -->
      <div class="title">LOGIN</div>

      <!-- Password Input -->
      <input
        type="password"
        disabled={!unlocked}
        bind:value={loginPassword}
        on:keydown={e => {
          if (e.key === 'Enter') { handleLogin() }
        }}
      />
      <button on:click={handleLogin} login>Submit</button>
    {:else}
      <div class="locked-icon"><Fa icon={faLock} size="8x" /></div>
      <div class="locked-text">{hours > 0 ? hours + " HOURS" : ""} {minutes > 0 ? minutes + " MINUTES" : ""} {seconds} SECONDS</div>
    {/if}
  </div>
{/if}

<style>
  .container {
    display: flex;
    gap: 20px;
    background-color: var(--bg-base);
    color: var(--text-color);
    margin: auto;
    border-radius: 16px;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }
  .wide {
    width: 992px;
    height: calc(100vh - 240px);
    margin-top: 120px;
    max-width: 98vw;
  }
  .slim {
    width: 100vw;
  }
  .title {
    font-size: 60px;
    font-family: var(--title-font);
  }
  input {
    width: 50%;
    line-height: 42px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    text-align: center;
  }
  input:active, :focus {
    outline: none; 
  }
  input:disabled {
    border-color: red;
  }
  button {
    background: var(--btn-special);
    color: var(--text-card-color);
    margin: 20px;
    padding: 12px 32px;
    border: none;
    border-radius: 16px;
  }
  button:hover {
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
  .info {
    margin: 20px;
    padding: 12px 0;
  }
  .locked-icon {
    color: var(--locked-color);
  }
  .locked-text {
    font-size: 42px;
    font-family: var(--title-font);
  }
</style>
