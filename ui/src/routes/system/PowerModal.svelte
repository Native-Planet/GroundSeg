<script>
  import { structure, connected } from '$lib/stores/data'
  import { afterUpdate } from 'svelte'
  import { shutdownDevice, restartDevice } from '$lib/stores/websocket'
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
        <div class="header">Shutdown</div>
        <div class="name">You are about to turn off your device. Continue?</div>
      {:else if info == "restart"}
        <div class="header">Restart</div>
        <div class="name">You are about to restart your device. Continue?</div>
      {/if}
      <button on:click={handleButton}>
        {info == "shutdown" ? "Shutdown" : "Restart"}
      </button>
    </div>
  {:else}
    <div class="wrapper transition-{info == "shutdown" ? "shutdown" : "restart"}">
      {#if info == "restart"}
        <div class="loader">
        </div>
      {/if}
      <div class="text">
        {info == "shutdown" ? "Device turned off" : "Restarting your device"}
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
    padding: 120px 0;
    margin: auto;
    text-align:center;
    font-size: 32px;
  }
  .transition-restart {
    display: flex;
    padding: 120px 0;
    font-size: 32px;
    justify-content: center;
    align-items: center;
    gap: 16px;
  }
  .loading-icon {
    width: 64px;
    height: 64px;
  }
  .loader {
    font-size:48px;
    color: var(--text-color);
    width: 1em;
    height: 1em;
    box-sizing: border-box;
    background-color: currentcolor;
    position: relative;
    border-radius: 50%;
    transform: rotateX(-60deg) perspective(1000px);
  }
  .loader:before,
  .loader:after {
    content: '';
    display: block;
    position: absolute;
    box-sizing: border-box;
    top: 0;
    left: 0;
    width: inherit;
    height: inherit;
    border-radius: inherit;
    animation: flowerFlow 1s ease-out infinite;
  }
.loader:after {
  animation-delay: .4s;
}

@keyframes flowerFlow {
  0% {
    opacity: 1;
    transform: rotate(0deg);
    box-shadow: 0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor,
    0 0 0 -.5em currentcolor;
  }
  100% {
    opacity: 0;
    transform: rotate(180deg);
    box-shadow: -1em -1em 0 -.35em currentcolor,
    0 -1.5em 0 -.35em currentcolor,
    1em -1em 0 -.35em currentcolor,
    -1.5em 0 0 -.35em currentcolor,
    1.5em -0 0 -.35em currentcolor,
    -1em 1em 0 -.35em currentcolor,
    0 1.5em 0 -.35em currentcolor,
    1em 1em 0 -.35em currentcolor;
  }
}
</style>
