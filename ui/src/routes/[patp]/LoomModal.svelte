<script>
  import { afterUpdate } from 'svelte'
  import { setUrbitLoom, structure } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let patp
  export let curLoomSize
  export let loomSize
  export let isOpen
  $: loomMB = (2**loomSize) / (1024*1024)
  $: curLoomMB = (2**curLoomSize) / (1024*1024)
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">Modify Urbit Loom</div>
    <div class="name">You are about to change your Urbit loom size from {loomMB} MB to {curLoomMB} MB</div>
    <button on:click={()=>setUrbitLoom(patp, curLoomSize)}>Modify</button>
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
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    max-width: 365px;
    margin: 64px 0;
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
  .transition-shutdown {
    background: var(--bg-card);
    padding: 120px 0;
    color: var(--text-card-color);
    font-size: 32px;
  }
  .transition-restart {
    background: var(--bg-card);
    padding: 120px 0;
    color: var(--text-card-color);
    font-size: 32px;
    animation: breathe 5s infinite;
  }
  @keyframes breathe {
    0% {
      background-color: #FFFFFF00;
      color: var(--text-color);
    }
    50% {
      background-color: var(--bg-card);
      color: var(--text-card-color);
    }
    100% {
      background-color: #FFFFFF00;
      color: var(--text-color);
    }
  }
</style>
