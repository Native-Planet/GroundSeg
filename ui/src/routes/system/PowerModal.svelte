<script>
  import { afterUpdate } from 'svelte'
  import { shutdownDevice, restartDevice, structure, connected } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'
  export let info

  let isConnected = $connected
  let isSend = false

  afterUpdate(()=> {
    if (isConnected && !$connected) {
      waitForConn()
    }
  })

  const waitForConn = () => {
    if (!$connected) {
      setTimeout(waitForConn, 1000)
    } else {
      closeModal()
    }
  }

  const handleButton = () => {
    isSend = !isSend
    if (info == "shutdown") {
      shutdownDevice()
    } else if (info == "restart") {
      restartDevice()
    }
  }
</script>

<Modal>
  {#if !isSend}
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
  {:else}
    <div class="wrapper transition-{info == "shutdown" ? "shutdown" : "restart"}">
      {#if info == "shutdown"}
        Device turned off
      {:else if info == "restart"}
        Restarting your device
      {/if}
    </div>
  {/if}
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
