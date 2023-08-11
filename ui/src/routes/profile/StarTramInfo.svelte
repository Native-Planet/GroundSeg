<script>
  import { structure, startramToggle, startramRestart } from '$lib/stores/websocket'
  import { showCancelModal, showRegisterModal, showEndpointModal } from './store'

  $: info = ($structure?.profile?.startram?.info) || {}
  $: registered = (info?.registered) || false
  $: region = (info?.region) || ""
  $: running = (info?.running) || ""
  $: expiry = (info?.expiry) || ""
  $: renew = (info?.renew) || ""
  $: endpoint = (info?.endpoint) || ""

  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tToggle = (transition?.toggle) || null
</script>
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
        {#if renew == "yes"}
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

<style>
  .body {
    display: flex;
    gap: 40px;
  }
  .left {
    flex: 8;
  }
  .spacer {
    flex: 1;
  }
  .right {
    flex: 5
  }
  .header {
    font-size: 12px;
    margin-bottom: 12px;
  }
  table {
    text-align: center;
    border: 2px solid var(--btn-secondary);
    padding: 20px;
    background-color: var(--bg-modal);
    border-radius: 12px;
    margin-bottom: 24px;
  }
  .top {
    font-size: 12px;
    line-height: 12px;
    padding-bottom: 4px;
  }
  .bottom {
    font-size: 24px;
    font-family: var(--title-font);
    line-height: 36px;
  }
  td {
    width: 200px;
  }
  .endpoint {
    padding: 12px;
    font-size: 12px;
    text-align: center;
    background-color: var(--bg-modal);
    border-radius: 12px;
    border: 2px solid var(--btn-secondary);
  }
  .btn-troubleshoot {
    width: calc(100% - 24px);
    padding: 12px;
    font-family: var(--regular-font);
    color: var(--text-card-color);
    font-size: 12px;
    border-radius: 12px;
    background: var(--btn-secondary);
    margin-bottom: 24px;
  }
  .unregistered-panel {
    margin-top: auto;
  }
  .btn-account {
    display: block;
    padding: 12px;
    width: calc(100% - 24px);
    font-family: var(--regular-font);
    color: var(--text-card-color);
    font-size: 12px;
    border-radius: 12px;
    background: var(--btn-secondary);
    margin-top: 8px;
  }
  /*
  input {
    width: calc(100% - 20px);
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 12px;
    font-size: 12px;
    line-height: 36px;
  }
  input:focus {
    outline: none;
  }
  */
</style>
