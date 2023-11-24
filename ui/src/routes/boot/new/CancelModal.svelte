<script>
  import { 
    structure,
    cancelNewShip,
  } from '$lib/stores/websocket'

  import { afterUpdate } from 'svelte'

  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  import { page } from '$app/stores'

  export let patp
  export let isOpen

  $: transition = ($structure?.urbits?.[patp]?.transition) || {}
  $: tDeleteShip = (transition?.deleteShip) || ""

  afterUpdate(()=>{
    if (tDeleteShip == "done") {
      closeModal()
    }
  })
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">Abort Booting New Ship:</div>
    <div class="patp"> {patp}</div>
    <div class="name">Are you sure?</div>
    <div class="button-wrapper">
      <button
        class="abort"
        disabled={tDeleteShip.length > 0}
        on:click={()=>cancelNewShip(patp)}>
        {#if tDeleteShip.length < 1}
          Yes, Abort
        {:else if tDeleteShip == "stopping"}
          Stopping boot process
        {:else if tDeleteShip == "removing-services"}
          Removing registered services
        {:else if tDeleteShip == "deleting"}
          Deleting local data
        {:else if tDeleteShip == "success"}
          {patp} boot aborted!
        {/if}
      </button>
    </div>
  </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .header {
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
  .patp {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 700;
    line-height: 64px; /* 200% */
    letter-spacing: -1.44px;
  }
  .name {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 42px; /* 133.333% */
    letter-spacing: -1.44px;
    max-width: 460px;
    margin: 32px 0 64px 0;
  }
  .button-wrapper {
    margin-top: 32px;
    display: flex;
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: var(--btn-secondary);
    border-radius: 16px;
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
    cursor: pointer;
    height: 65px;
  }
  button:disabled {
    opacity: .6;
    pointer-events:none;
  }
  .abort {
    background: black;
  }
</style>
