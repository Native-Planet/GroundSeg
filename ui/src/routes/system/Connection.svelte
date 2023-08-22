<script>
  import { afterUpdate } from 'svelte'
  import { structure, toggleWifi, connectWifi } from '$lib/stores/websocket'
  import ToggleButton from '$lib/ToggleButton.svelte'

  import Fa from 'svelte-fa'
  import { faAngleUp, faAngleDown } from '@fortawesome/free-solid-svg-icons';

  let showNetworks = false
  let toChange = false
  let selectedNetwork = ""
  let password = ''

  $: wifi = ($structure?.system?.wifi) || {}
  $: status = (wifi?.status) || false
  $: active = (wifi?.active) || null
  $: networks = (wifi?.networks) || []

  afterUpdate(()=>{
    if (!status) {
      if (showNetworks) {
        showNetworks = false
      }
    }
    if (selectedNetwork == active) {
      toChange = false
    }
  })

  // select new network
  const selectNetwork = network => {
    toChange = true
    selectedNetwork = network
    showNetworks = false
  }
</script>

<div class="container">
  <div class="title">CONNECTION</div>

  <div class="wifi-toggle">
    <div class="wifi-text">Wi-Fi</div>
    <ToggleButton
      on:click={toggleWifi}
      on={status}
      />
  </div>

  <div class="wifi-options">
    {#if status}
      <div class="active">
        <div class="active-title">Network Name</div>
        <div class="active-selector" on:click={()=>showNetworks = !showNetworks}>
          {#if toChange}
            <div class="active-text">{selectedNetwork}</div>
          {:else}
            <div class="active-text">{active == null ? "Select a wireless network" : active}</div>
          {/if}

          <div class="active-arrow">
            {#if showNetworks}
              <Fa icon={faAngleUp} size="1x" />
            {:else}
              <Fa icon={faAngleDown} size="1x" />
            {/if}
          </div>
        </div>

      </div>

      {#if showNetworks}
        <div class="networks">
          {#each networks as n}
            <div class="network" on:click={()=>{selectNetwork(n)}}>{n}</div>
          {/each}
        </div>
      {/if}

      {#if toChange}
        <div class="submit">
          <input type="password" placeholder="Wi-Fi Password" bind:value={password} />
          <div class="submit-buttons">
            <button class="cancel" on:click={()=>toChange = false}>Cancel</button>
            <button disabled={password.length < 1} class="connect" on:click={()=>connectWifi(selectedNetwork,password)}>Connect</button>
          </div>
        </div>
      {/if}

    {/if}
  </div>

</div>

<style>
  .container {
    margin: 0;
  }
  .title {
    margin-bottom: 20px;
  }
  .wifi-toggle {
    display: flex;
    align-items: center;
    margin-bottom: 12px;
  }
  .wifi-options {
    display: flex;
    flex-direction: column;
  }
  .wifi-text {
    flex: 1;
    font-size: 16px;
    font-weight: 500;
  }
  .active-title {
    font-size: 13px;
    margin-bottom: 12px;
  }
  .active-selector {
    display: flex;
    background: var(--bg-modal);
    padding: 8px 0 8px 24px;
    align-items: center;
    border-radius: 8px;
  }
  .active-text {
    flex: 1;
    font-size: 13px;
    font-weight: 600;
  }
  .active-arrow {
    width: 40px;
  }
  .networks {
    display: flex;
    flex-direction: column;
    gap: 20px;
    background: var(--btn-secondary);
    padding: 20px;
    color: var(--text-card-color);
    border-radius: 8px;
  }
  .submit {
    margin-top: 20px;
    display: flex;
    flex-direction: column;
  }
  input {
    padding: 8px 20px;
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 8px;
    font-size: 13px;
  }
  input:focus {
    outline: none;
  }
  .submit-buttons {
    margin-top: 20px;
    display: flex;
    gap: 24px;
  }
  button {
    border-radius: 12px;
    color: var(--text-card-color);
    padding: 12px 24px;
    font-family: var(--regular-font);
    font-size: 12px;
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .cancel {
    background-color: var(--btn-secondary);
  }
  .connect {
    background-color: var(--btn-primary);
  }
</style>
