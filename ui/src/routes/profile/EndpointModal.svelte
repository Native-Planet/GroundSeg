<script>
  import { structure, startramEndpoint } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let isOpen

  let success = false
  let newEndpoint = ''

  $: info = ($structure?.profile?.startram?.info) || {}
  $: endpoint = (info?.endpoint) || "api.startram.io"
</script>

{#if isOpen}
<Modal>
  <div class="wrapper">
    <!-- Info -->
    <h1>Edit Endpoint</h1>
    <p>Modifying your endpoint removes previous StarTram configuration</p>

    <!-- Replacement Endpoint -->
    <h2>New Endpoint</h2>
    <input placeholder="example.endpoint.com" type="text" bind:value={newEndpoint} />

    <button
      disabled={(newEndpoint.length < 1) || (newEndpoint == endpoint)} 
      on:click={()=>startramEndpoint(newEndpoint)}
      >Set New Endpoint
    </button>
  </div>
</Modal>
{/if}

<style>
  .wrapper {
    margin: 20px;
    font-family: var(--regular-font);
  }
  h1 {
    font-size: 14px;
    font-weight: 500;
  }
  p {
    margin-top: 10px;
    font-size: 14px;
    font-weight: 300;
    margin-bottom: 20px;
    opacity: .8;
  }
  h2 {
    font-size: 14px;
    font-weight: 300;
    margin: 0;
    color: var(--btn-secondary);
    font-weight: 500;
  }
  input {
    font-size: 14px;
    margin: 8px 0 20px 0;
    background: var(--bg-modal);
    border-radius: 12px;
    width: calc(100% - 24px);
    border: none;
    padding: 12px;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  button {
    margin-top: 48px;
    background-color: var(--btn-secondary);
    border-radius: 12px;
    color: var(--text-card-color);
    height: 42px;
    padding: 0 64px;
    font-family: var(--regular-font);
    font-size: 12px;
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
