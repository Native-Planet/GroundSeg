<script>
  /*
  import { goto } from '$app/navigation'
  import { afterUpdate } from 'svelte'
  import { deleteUrbitShip } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import Sigil from './Sigil.svelte'
*/
  import Fa from 'svelte-fa'
  import { faPlugCircleExclamation } from '@fortawesome/free-solid-svg-icons';

  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  export let component
  export let isOpen


  /*
  $: transition = ($structure?.urbits?.[patp]?.transition) || {}
  $: tDeleteShip = (transition?.deleteShip) || ""
  */

  let text;

  $:if (component == "power") {
    text = "Shuts off ship's power. Reconnect via alternative access to GroundSeg"
  } else if (component == "meld") {
    text = "Temporarily disconnects, runs ship in Meld mode, then auto-reconnects."
  } else if (component == "remote") {
    text = "Blocks access from current domain. Use new domain for reconnection."
  } else if (component == "dev") {
    text = "Causes temporary disconnection; ship reboots in developer mode."
  } else if (component == "loom") {
    text = "Temporarily disconnects; ship restarts with new loom size."
  } else if (component == "rebuild") {
    text = "Temporarily disconnects during container rebuild; auto-reconnects afterward."
  } else {
    text = "How did you get here?"
  }
</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="text">
      <Fa icon={faPlugCircleExclamation} size="1x" />
      Ship Connection Loss If Modified
    </div>
    <div class="text sub">{text}</div>
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
