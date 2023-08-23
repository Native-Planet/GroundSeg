<script>
  import { structure,  startramRestart } from '$lib/stores/websocket'
  import { showCancelModal, showRegisterModal, showEndpointModal } from './store'
  export let renew
  export let expiry
</script>

<div class="wrapper">
  <div class="title">STARTRAM</div>
  <div class="info-wrapper">
    <div class="info">
      <div class="item">Autorenew</div>
      <div class="item">{renew ? "Yes" : "No"}</div>
    </div>
    <div class="info">
      <div class="item">Expiration Date</div>
      <div class="item">{expiry}</div>
    </div>
  </div>
</div>

<style>
  .wrapper {
    width: 30%;
  }
  .title {
    margin-bottom: 12px;
  }
  .info-wrapper {
    display: flex;
    margin-bottom: 12px;
  }
  .info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .item {
    font-size: 14px;
  }
</style>

<!--
<div class="body">
  <div class="panel left">
    <div class="header">Subscription Information</div>
    <table>
      {#if registered}
        <tr class="top">
          <td>Active Region</td>
          <td>{renew ? "Renewal" : "Expiry"} Date</td>
          <td>Auto Renewal</td>
        </tr>
        <tr class="bottom">
          <td>{region.toUpperCase()}</td>
          <td>{expiry}</td>
          <td>{renew ? "Yes":"No"}</td>
        </tr>
      {:else}
        <tr class="top">
          <td></td>
          <td>Not Registered</td>
          <td></td>
        </tr>
      {/if}
    </table>
    <div class="header">Current Endpoint</div>
    <div class="endpoint">{endpoint}</div>
  </div>
  <div class="spacer"></div>
  {#if registered}
    <div class="panel right">
      <div class="header">Troubleshoot</div>
      <button on:click={startramToggle} class="btn-troubleshoot">
        {#if tToggle == "loading"}
          Loading..
        {:else}
          Turn {running ? "Off" : "On"}
        {/if}
      </button>
      <button on:click={startramRestart} class="btn-troubleshoot">Restart StarTram</button>
      <div class="header">Account</div>
      <div class="account">
        <button on:click={()=>showRegisterModal.set(true)} class="btn-account">Register Another Key</button>
        <button on:click={()=>showEndpointModal.set(true)} class="btn-account">Modify Endpoint</button>
        {#if registered}
          <button on:click={()=>showCancelModal.set(true)} class="btn-account">Cancel Subscription</button>
        {/if}
      </div>
    </div>
  {:else}
    <div class="unregistered-panel right">
      <div class="account">
        <button on:click={()=>showRegisterModal.set(true)} class="btn-account">Register Key</button>
        <button on:click={()=>showEndpointModal.set(true)} class="btn-account">Modify Endpoint</button>
      </div>
    </div>
  {/if}
</div>
-->
