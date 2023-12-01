<script>
  import { afterUpdate } from 'svelte'
  import { toggleWifi, connectWifi } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import ToggleButton from '$lib/ToggleButton.svelte'

  import Fa from 'svelte-fa'
  import { faAngleUp, faAngleDown } from '@fortawesome/free-solid-svg-icons';

  let showNetworks = false
  let toChange = false
  let selectedNetwork = ""
  let password = ''

  $: wifi = ($structure?.system?.info?.wifi) || {}
  $: status = (wifi?.status) || false
  $: active = (wifi?.active) || ""
  $: networks = (wifi?.networks) || []

  $: tWifiConnect = ($structure?.system?.transition?.wifiConnect) || ""

  afterUpdate(()=>{
    if (!status) {
      if (showNetworks) {
        showNetworks = false
      }
    }
    if (selectedNetwork == active) {
      toChange = false
    }
    if (!toChange) {
      password = ''
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
  <div class="sys-title">CONNECTION</div>

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
        <div class="active-selector" class:disabled={tWifiConnect.length > 0} on:click={()=>showNetworks = !showNetworks}>
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
        <div class="networks" class:disabled={tWifiConnect.length > 0}>
          {#each networks as n}
            <div class="network" on:click={()=>{selectNetwork(n)}}>{n}</div>
          {/each}
        </div>
      {/if}

      {#if toChange}
        <div class="submit">
          <input disabled={tWifiConnect.length > 0} type="password" placeholder="Wi-Fi Password" bind:value={password} />
          <div class="submit-buttons">
            <button class="cancel" on:click={()=>toChange = false} disabled={tWifiConnect.length > 0}>Cancel</button>
            <button disabled={(password.length < 1) || tWifiConnect.length > 0} class="connect" on:click={()=>connectWifi(selectedNetwork,password)}>
              {#if tWifiConnect.length < 1}
                Connect
              {:else if tWifiConnect == "connecting"}
                Attempting to connect..
              {:else if tWifiConnect == "success"}
                Connected!
              {:else if tWifiConnect == "error"}
                Failed due to invalid credentials!
              {/if}
            </button>
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
  .sys-title {
    margin-bottom: 56px;
  }
  .wifi-toggle {
    display: flex;
    align-items: center;
  }
  .wifi-options {
    display: flex;
    flex-direction: column;
  }
  .wifi-text {
    flex: 1;
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .active-title {
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
    margin-top: 32px;
  }
  .active-selector {
    display: flex;
    background: var(--bg-modal);
    align-items: center;
    border-radius: 16px;
    height: 65px;
    cursor: pointer;
  }
  .active-text {
    flex: 1;
    font-size: 13px;
    font-weight: 600;

    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 20px;
  }
  .active-arrow {
    width: 40px;
  }
  .networks {
    margin-top: 16px;
    display: flex;
    flex-direction: column;
    background: var(--btn-secondary);
    padding: 20px 0;
    color: var(--text-card-color);
    border-radius: 16px;
  }
  .network {
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 10px 20px;
  }
  .network:hover {
    background: var(--bg-card);
    
  }
  .submit {
    margin-top: 32px;
    display: flex;
    flex-direction: column;
  }
  input {
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 16px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 0 20px 0 20px;
    height: 61px;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .submit-buttons {
    margin-top: 20px;
    display: flex;
    gap: 24px;
  }
  button {
    border-radius: 16px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    padding: 0 48px;
    color: var(--text-card-color);
    font-family: var(--regular-font);
    cursor: pointer;
    height: 65px;
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
  .disabled {
    opacity:.6;
    pointer-events: none;
  }
</style>
