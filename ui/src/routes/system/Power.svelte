<script>
  // Modals
  import { openModal } from 'svelte-modals'
  import PowerModal from './PowerModal.svelte'
  // Websocket
  import { structure } from '$lib/stores/websocket'
</script>

<div class="container">
  <div class="sys-title">POWER</div>
  <div class="spacer"></div>
  <button class="btn" on:click={()=>openModal(PowerModal,{"info":"shutdown"})}>Shut Down</button>
  <button class="btn" on:click={()=>openModal(PowerModal,{"info":"restart"})}>Restart</button>
</div>

<style>
  .container {
    margin: 0;
    display: flex;
    gap: 24px;
    align-items: center;
  }
  .sys-title {
    margin: 0;
  }
  .spacer {
    flex: 1;
  }
  .btn {
    height: 56px;
    border-radius: 16px;
    cursor: pointer;
    padding: 0 48px;
    background: var(--btn-secondary);

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
</style>
