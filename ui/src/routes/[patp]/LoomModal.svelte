<script>
  import { afterUpdate } from 'svelte'
  import { setUrbitLoom } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let patp
  export let curLoomSize
  export let loomSize
  export let isOpen
  $: loomMB = (2**loomSize) / (1024*1024)
  $: curLoomMB = (2**curLoomSize) / (1024*1024)
  $: tLoom = ($structure?.urbits?.[patp]?.transition?.loom) || ""

  afterUpdate(()=>{
    if (tLoom == "done") {
      closeModal()
    }

  })
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">Modify Urbit Loom</div>
    <div class="name">You are about to change your Urbit loom size from {loomMB} MB to {curLoomMB} MB</div>
    <button disabled={tLoom.Length > 0} on:click={()=>setUrbitLoom(patp, curLoomSize)}>
      {#if tLoom.length < 1}
        Modify
      {:else if tLoom == "loading"}
        Modifying..
      {:else if tLoom == "success"}
        Modified!
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
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
