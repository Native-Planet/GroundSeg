<script>
  import { goto } from '$app/navigation'
  import { afterUpdate } from 'svelte'
  import { deleteUrbitShip } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import Sigil from './sigil.svelte'

  import Modal from '$lib/modal.svelte'
  import { closeModal } from 'svelte-modals'

  export let patp
  export let isOpen

  $: transition = ($structure?.urbits?.[patp]?.transition) || {}
  $: tDeleteShip = (transition?.deleteShip) || ""

  afterUpdate(()=>{
    if (tDeleteShip == "done") {
      closeModal()
      goto("/")
    }
  })
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">Delete Urbit Ship</div>
    <Sigil name={patp} modal={true} />
    <div class="name">You are attempting to delete all data related to <strong>{patp}</strong>.</div>
    <button disabled={tDeleteShip.length > 0} on:click={()=>deleteUrbitShip(patp)}>
      {#if tDeleteShip.length < 1}
        Delete
      {:else if tDeleteShip == "stopping"}
        Stopping the ship
      {:else if tDeleteShip == "removing-services"}
        Removing StarTram services
      {:else if tDeleteShip == "deleting"}
        Deleting local data
      {:else if tDeleteShip == "success"}
        {patp} deleted!
      {/if}
    </button>
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
    margin: 32px 0;
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: #000;
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
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
