<script>
  import { 
    exportUrbitShip,
    exportUrbitBucket
  } from '$lib/stores/websocket'

  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  export let patp
  export let isOpen
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">Export For {patp}</div>
    <div class="name">What do you want to export?</div>
    <div class="button-wrapper">
      <button on:click={()=>exportUrbitShip(patp)}>Urbit Ship</button>
      <button on:click={()=>exportUrbitBucket(patp)}>Storage</button>
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
  .button-wrapper {
    display: flex;
    gap: 48px;
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
</style>
