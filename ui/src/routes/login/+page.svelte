<script>
  import { login, loginError } from '$lib/stores/websocket'
  import { isSessionRemembered } from '$lib/stores/gs-crypto'
  import { structure } from '$lib/stores/data'
  import { wide } from '$lib/stores/display'
  import { onMount, onDestroy } from 'svelte'

  import Fa from 'svelte-fa'
  import { faLock } from '@fortawesome/free-solid-svg-icons'

  let passwordInput;
  let inView = false
  let loginPassword = ''
  let rememberMe = false

  $: if ($loginError) {
    showModal($loginError);
  }

  $: loginModule = ($structure?.login) || null
  $: remainder = (loginModule?.remainder) || 0
  $: unlocked = (remainder <= 0)

  $: hours = Math.floor(remainder / 3600)
  $: minutes = Math.floor((remainder % 3600) / 60)
  $: seconds = Math.floor(remainder % 60)

  onMount(() => {
    rememberMe = isSessionRemembered()
    inView = true;
    setTimeout(() => {
      if (passwordInput && unlocked) {
        passwordInput.focus();
      }
    }, 100); // needs short timeout
  });
  onDestroy(() => inView = false);

  const handleLogin = async () => {
    login(loginPassword, rememberMe)
  }

  function showModal(message) {
    setTimeout(() => {
      loginError.set('');
    }, 2000);
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
      <div class="pw-wrapper">
        <input
          class="password-input"
          bind:this={passwordInput}
          type="password"
          disabled={!unlocked}
          bind:value={loginPassword}
          on:keydown={e => {
            if (e.key === 'Enter') { handleLogin() }
          }}
        />
      </div>
      <label class="remember-row">
        <input class="remember-input" type="checkbox" bind:checked={rememberMe} />
        <span class="remember-box" aria-hidden="true">
          {#if rememberMe}
            <img class="checkmark" src="/checkmark.svg" alt="" />
          {/if}
        </span>
        <span class="remember-text">Remember me</span>
      </label>
      <button on:click={handleLogin} login>Submit</button>
    {:else}
      <div class="locked-icon"><Fa icon={faLock} size="8x" /></div>
      <div class="locked-text">{hours > 0 ? hours + " HOURS" : ""} {minutes > 0 ? minutes + " MINUTES" : ""} {seconds} SECONDS</div>
    {/if}
    {#if $loginError}
      <div class="modal">
        {$loginError}
      </div>
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
    margin-top: 60px;
    max-width: 100vw;
  }
  .slim {
    width: 100vw;
  }
  .title {
    font-size: 60px;
    font-family: var(--title-font);
  }
  .password-input {
    flex: 1;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    border: none;
    height: 30px;
    width: 480px;
    max-width: 80vw;
    text-align: center;
    padding: 32px 24px 36px 24px;
  }
  .password-input:focus {
    outline: none;
  }
  .remember-row {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 480px;
    max-width: 80vw;
    cursor: pointer;
    user-select: none;
  }
  .remember-input {
    position: absolute;
    width: 1px;
    height: 1px;
    opacity: 0;
  }
  .remember-box {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    border: solid 2px var(--text-color);
    border-radius: 8px;
    background: var(--Gray-100, #DDE3DF);
    flex-shrink: 0;
  }
  .remember-input:focus-visible + .remember-box {
    outline: solid 2px var(--btn-special);
    outline-offset: 3px;
  }
  .remember-text {
    color: var(--Gray-400, #5C7060);
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: 0;
  }
  .checkmark {
    width: 16px;
    height: 16px;
  }
  button {
    margin-top: 32px;
    font-family: var(--regular-font);
    background: var(--btn-special);
    color: var(--text-card-color);
    height: 65px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
    justify-content: center;
    align-items: center;
    padding: 0 48px;
    cursor: pointer;

    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
  }
  .locked-icon {
    color: var(--locked-color);
  }
  .locked-text {
    font-size: 42px;
    font-family: var(--title-font);
  } 
  .modal {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    background-color: white;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    z-index: 1000;
    animation: fadeInOut 0.2s ease-in-out;
  }
  @keyframes fadeInOut {
    0%, 100% { opacity: 0; }
    10%, 90% { opacity: 1; }
  }
</style>
