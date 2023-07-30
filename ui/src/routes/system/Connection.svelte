<script>
  import { afterUpdate } from 'svelte'
  import { structure, toggleWifi, connectWifi } from '$lib/stores/websocket'

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
    <div class="btn-wifi" on:click={toggleWifi}>{status ? "on" : "off"}</div>
  </div>
  <div class="wifi-options">
    {#if status}
      <div class="active">

        {#if toChange}
          <div class="active-text">{selectedNetwork}</div>
        {:else}
          <div class="active-text">{active == null ? "Select a wireless network" : active}</div>
        {/if}

        <div class="active-arrow" on:click={()=>showNetworks = !showNetworks}>
          {#if showNetworks}
            <Fa icon={faAngleUp} size="1x" />
          {:else}
            <Fa icon={faAngleDown} size="1x" />
          {/if}
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
          <input type="password" placeholder="wifi password" bind:value={password} />
          <div class="submit-buttons">
            <button class="cancel" on:click={()=>toChange = false}>Cancel</button>
            <button class="connect" on:click={()=>connectWifi(selectedNetwork,password)}>Connect</button>
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
  .wifi-toggle {
    display: flex;
  }
  .wifi-options {
    display: flex;
    flex-direction: column;
    gap: 30px;
  }
  .wifi-text {
    flex: 1;
    font-size: 14px;
  }
  .active {
    display: flex;
  }
  .active-text {
    flex: 1;
  }
  .active-arrow {
    width: 40px;
  }
  .networks {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
</style>
