<script>
  import { afterUpdate } from 'svelte'
  import { structure, startramEndpoint } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let isOpen

  let success = false
  let newEndpoint = ''

  $: endpoint = ($structure?.profile?.startram?.info?.endpoint) || ""
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tEndpoint = (transition?.endpoint) || ""

  let completed = false

  afterUpdate(()=>{
    if (completed) {
      if (tEndpoint.length < 1) {
        closeModal()
      }
    } else {
      if (tEndpoint == "complete") {
        completed = true
      }
    }
  })
</script>

{#if isOpen}
<Modal>
  {#if tEndpoint.length < 1}
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
  {:else if tEndpoint == "init"}
    <div class="notify">Loading</div>
  {:else if tEndpoint == "unregistering"}
    <div class="notify">Removing old configuration</div>
  {:else if tEndpoint == "stopping"}
    <div class="notify">Stopping StarTram</div>
  {:else if tEndpoint == "configuring"}
    <div class="notify">Applying new configuration</div>
  {:else if tEndpoint == "finalizing"}
    <div class="notify">Finishing up</div>
  {:else if tEndpoint == "complete"}
    <div class="success">Successfully edited endpoint!</div>
  {:else}
    {tEndpoint}
  {/if}
</Modal>
{/if}

<style>
  .notify {
    padding: 120px 0px;
    font-size: 24px;
    text-align: center;
    animation: breathe 5s infinite;
    color: var(--btn-secondary);
  }
  .success {
    padding: 120px 0px;
    font-size: 24px;
    text-align: center;
    color: var(--btn-primary);
  }
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
