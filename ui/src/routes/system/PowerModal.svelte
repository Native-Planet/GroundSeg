<script>
  import { shutdownDevice, restartDevice } from '$lib/stores/websocket'
  import Modal from '$lib/Modal.svelte'
  export let info

  const handleButton = () => {
    if (info == "shutdown") {
      shutdownDevice()
    } else if (info == "restart") {
      restartDevice()
    }
  }
</script>

<Modal>
  <div class="wrapper">
    {#if info == "shutdown"}
      <div class="header">TURN OFF DEVICE</div>
      <div class="name">You are about to turn off your device. Continue?</div>
    {:else if info == "restart"}
      <div class="header">RESTART DEVICE</div>
      <div class="name">You are about to restart your device. Continue?</div>
    {/if}
    <button on:click={handleButton}>
      {info == "shutdown" ? "Shutdown" : "Restart"}
    </button>
  </div>
</Modal>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    border-radius: 16px;
    padding: 40px;
    gap: 24px;
  }
  .header {
    font-family: var(--title-font);
    font-size: 24px;
  }
  .name {
    width: 100%;
    padding: 16px 0;
    font-family: var(--regular-font);
    font-size: 12px;
    background: var(--bg-warning);
    border-radius: 12px;
    text-align: center;
    font-weight: 600;
  }
  button {
    background: black;
    font-family: var(--regular-font);
    color: var(--text-card-color);
    flex: 1;
    padding: 12px 48px;
    border-radius: 16px;
  }
  button:hover {
    cursor: pointer;
  }
</style>
