<script>
  import { afterUpdate } from 'svelte'
  import { setUrbitSnapTime } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  import { URBIT_MODE } from '$lib/stores/data'
  export let patp
  export let curSnapTime
  export let snapTime
  export let isOpen
  $: tSnapTime = ($structure?.urbits?.[patp]?.transition?.snapTime) || ""

  afterUpdate(()=>{
    if (tSnapTime == "done") {
      closeModal()
    }
  })
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="header">Modify Snapshot Time</div>
    {#if $URBIT_MODE}
      <div class="sub">You are currently accesing GroundSeg through this ship. You will temporarily lose access if you continue.</div> 
    {/if}
    <div class="name">You are about to change your snapshot interval from {snapTime} seconds to {curSnapTime} seconds</div>
    <button disabled={tSnapTime.length > 0} on:click={()=>setUrbitSnapTime(patp, curSnapTime)}>
      {#if tSnapTime.length < 1}
        Modify
      {:else if tSnapTime == "loading"}
        Modifying..
      {:else if tSnapTime == "success"}
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
    line-height: 48px;
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
    line-height: 32px;
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
    line-height: 32px;
    letter-spacing: -1.44px;
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .sub {
    line-height: 32px;
    font-size: 20px;
    font-weight: 500;
    margin-top: 32px;
    color: var(--text-color);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    letter-spacing: -1.44px;
  }
</style>