<script>
  // Modals
  import { openModal } from 'svelte-modals'
  import EndpointModal from './EndpointModal.svelte'

  import { structure } from '$lib/stores/websocket'

  //$: info = ($structure?.profile?.startram?.info) || {}
  //$: endpoint = (info?.endpoint) || ""
</script>

<div class="wrapper">
  <div class="item">
    <div class="label">Endpoint</div>
    <div class="data">
      <div class="data-text">api.startram.io</div>
      <button
      on:click={()=>openModal(EndpointModal)}
      >Edit</button>
    </div>
  </div>

  <div class="item">
    <div class="label">Region</div>
    <div class="data">
      <div class="data-text">US East</div>
      <button>Edit</button>
    </div>
  </div>
</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    margin-top: 64px;
    gap: 32px;
  }
  .label {
    font-size: 17px;
    margin-bottom: 21px;
  }
  .data {
    display: flex;
    gap: 21px;
  }
  .data-text {
    flex: 1; 
    padding: 24px;
    background: var(--bg-modal);
    border-radius: 16px;
    font-size: 24px;
  }
  button {
    font-size: 24px;
    padding: 24px 48px;
    background: var(--btn-secondary);
    color: var(--text-card-color);
    border-radius: 16px;
    font-weight: 400;
  }
</style>
