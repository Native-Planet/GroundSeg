<script>
  import { openModal } from 'svelte-modals'
  import FinalModal from './FinalModal.svelte';
  import ToggleButton from '$lib/ToggleButton.svelte'
  import UnplugWarning from './UnplugWarning.svelte';
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  import Fa from 'svelte-fa'
  import { faPlugCircleExclamation } from '@fortawesome/free-solid-svg-icons';

 import { structure } from '$lib/stores/data'
  import { URBIT_MODE } from '$lib/stores/data'

  $: wgRunning = ($structure?.profile?.startram?.info?.running) || false

  export let patp
  export let remote
  export let remoteReady
  export let tToggleNetwork = ""
  export let ownShip

  const dispatch = createEventDispatcher()

  function handleClick() {
    if ($URBIT_MODE) {
      openModal(FinalModal, {"component":"remote","patp":patp})
    } else {
      dispatch("click")
    }
  }
</script>

<div class="section" class:disabled={!wgRunning || !remoteReady}>
  <div class="section-left">
    <div class="section-title">Remote Access</div>
    <div class="section-description">Access your ship via a StarTram connection</div>
  </div>
  <div class="section-right">
    <UnplugWarning component={"remote"} {ownShip}>
      <ToggleButton
        on:click={handleClick}
        on={remote}
        loading={tToggleNetwork.length > 0}
        />
    </UnplugWarning>
  </div>
</div>

<style>
  .disabled {
    opacity: .4;
    pointer-events: none;
  }
</style>
