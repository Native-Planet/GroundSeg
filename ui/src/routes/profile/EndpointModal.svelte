<script>
  import { afterUpdate } from 'svelte'
  import { startramEndpoint } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
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
      <p>Modifying your endpoint removes previous StarTram configuration.</p>

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
    margin: 32px;
    font-family: var(--regular-font);
  }
  h1 {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  p {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
    width: 500px;
  }
  h2 {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.2px;
  }
  input {
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
    padding: 16px 24px 18px 24px;
    width: calc(100% - 48px);
    border: none;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  button {
    margin-top: 56px;
    background-color: var(--btn-secondary);
    border-radius: 16px;
    cursor: pointer;
    padding: 0 48px;
    height: 65px;
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
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
