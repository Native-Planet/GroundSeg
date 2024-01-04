<script>
  import Fa from 'svelte-fa'
  import { faPlugCircleExclamation } from '@fortawesome/free-solid-svg-icons';
  import { URBIT_MODE } from '$lib/stores/data';

  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  import { 
    rebuildContainer,
    urthPackMeld,
    toggleUrbitPower,
    toggleDevMode,
    toggleNetwork,
  } from '$lib/stores/websocket'

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  export let patp
  export let component
  export let isOpen

  let text;
  let title;
  let btnText;
  let btnDisabled;

  // Text
  $:if (component == "power") {
    title = "Turn off ship"
    text = "You are currently accesing GroundSeg through this ship. You will lose access if you continue. Proceed?"
    btnText = "Proceed"
  } else if (component == "meld") {
    title = "Pack and meld ship from Earth"
    text = "You are currently accesing GroundSeg through this ship. You will temporarily lose access if you continue. Proceed?"
    btnText = "Proceed"
  } else if (component == "remote") {
    title = "Switch network"
    text = "You are currently accesing GroundSeg through this ship. You will lose access from this domain if you continue. Proceed?"
    btnText = "Proceed"
  } else if (component == "dev") {
    title = "Modify developer mode status"
    text = "You are currently accesing GroundSeg through this ship. You will temporarily lose access if you continue. Proceed?"
    btnText = "Proceed"
  } else if (component == "loom") {
    title = "Resize ship loom"
    text = "You are currently accesing GroundSeg through this ship. You will temporarily lose access if you continue. Proceed?"
    btnText = "Proceed"
  } else if (component == "rebuild") {
    title = "Resize ship loom"
    text = "You are currently accesing GroundSeg through this ship. You will temporarily lose access if you continue. Proceed?"
    btnText = "Proceed"
  } else {
    text = "How did you get here?"
  }

  // Action
  function handleAction() {
    btnText = "Loading"
    btnDisabled = true
    if (component == "power") {
      toggleUrbitPower(patp)
      setTimeout(()=>window.location.href(pfx+"/", 15000))
    } else if (component == "meld") {
      urthPackMeld(patp)
    } else if (component == "remote") {
      toggleNetwork(patp)
    } else if (component == "dev") {
      toggleDevMode(patp)
    } else if (component == "rebuild") {
      rebuildContainer(patp)
    }
  }
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="text">
      <Fa icon={faPlugCircleExclamation} size="1x" />
      {title}
    </div>
    <div class="text sub">{text}</div>
    <button disabled={btnDisabled} on:click={handleAction}>{btnText}</button>
  </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .text {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .sub {
    line-height: 32px;
    font-size: 20px;
    font-weight: 500;
    margin-top: 32px;
    margin-bottom: 32px;
    color: var(--text-color);
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
