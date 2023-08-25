<script>
  import { shutdownDevice, restartDevice } from '$lib/stores/websocket'
  import { shutdownModal, restartModal } from './store'
  export let info
</script>

<div class="wrapper">
  <div class="modal">
    {#if info == "shutdown"}
      <div class="header">TURN OFF DEVICE</div>
      <div class="name">You are about to turn off your device. Continue?</div>
      <div class="buttons">
        <button
          class="btn-cancel"
          on:click={()=>shutdownModal.set(false)}
          >Back
        </button>
        <button
          class="btn-activate"
          on:click={shutdownDevice}
          >Shutdown
        </button>
      </div>
    {:else if info == "restart"}
      <div class="header">RESTART DEVICE</div>
      <div class="name">You are about to restart your device. Continue?</div>
      <div class="buttons">
        <button
          class="btn-cancel"
          on:click={()=>restartModal.set(false)}
          >Back
        </button>
        <button
          class="btn-activate"
          on:click={restartDevice}
          >Restart
        </button>
      </div>
    {/if}

  </div>
</div>

<style>
  .wrapper {
    position:fixed;
    left: 0;
    top: 0;
    right: 0;
    bottom: 0;
    backdrop-filter: blur(4px);
    -moz-backdrop-filter: blur(4px);
    -o-backdrop-filter: blur(2px);
    -webkit-backdrop-filter: blur(4px);
    background: #FFFFFF3D;
  }
  .modal {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: absolute;
    top: calc(50vh - (392px/2));
    left: calc(50vw - (572px/2));
    background: var(--bg-modal);
    width: calc(572px - 80px);
    height: calc(392px - 80px);
    border-radius: 16px;
    padding: 40px;
    gap: 24px;
  }
  .header {
    font-family: var(--title-font);
    font-size: 24px;
  }
  .name {
    font-family: var(--regular-font);
    font-size: 12px;
    background: var(--bg-warning);
    padding: 40px;
    border-radius: 16px;
  }
  .btn-activate {
    background: var(--btn-primary);
    padding: 0 20px;
    color: var(--text-card-color);
    border-radius: 12px;
  }
  .btn-cancel {
    background: var(--btn-secondary);
    padding: 8px 20px;
    color: var(--text-card-color);
    border-radius: 12px;
  }
  .btn-activate:disabled {
    background: var(--btn-secondary);
    color: var(--text-color);
    opacity: .6;
  }
  .buttons {
    margin-top: 30px;
    display: flex;
    height: 36px;
    gap: 20px;
    width: 360px;
  }
  button:hover {
    cursor: pointer;
  }
  button {
    font-family: var(--regular-font);
    flex: 1;
  }
</style>
