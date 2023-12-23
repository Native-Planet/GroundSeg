<script>
  import { openModal } from 'svelte-modals'
  import FinalModal from './FinalModal.svelte';
  import ToggleButton from '$lib/ToggleButton.svelte'
  import UnplugWarning from './UnplugWarning.svelte';
  import { URBIT_MODE } from '$lib/stores/data'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  export let patp
  export let devMode
  export let tToggleDevMode = ""
  export let ownShip

  const dispatch = createEventDispatcher()

  function handleClick() {
    if ($URBIT_MODE) {
      openModal(FinalModal, {"component":"dev","patp":patp})
    } else {
      dispatch("click")
    }
  }
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Developer Mode</div>
    <div class="section-description">This enables your ship to be able to debug from another computer</div>
  </div>
  <div class="section-right">
    <UnplugWarning component={"dev"} {ownShip}>
      <ToggleButton
        on:click={handleClick}
        loading={tToggleDevMode.length > 0}
        on={devMode}
        />
    </UnplugWarning>
  </div>
</div>
