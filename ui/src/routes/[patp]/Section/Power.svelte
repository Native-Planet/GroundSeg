<script>
  import ToggleButton from '$lib/ToggleButton.svelte'
  import { openModal } from 'svelte-modals'
  import FinalModal from './FinalModal.svelte';
  import UnplugWarning from './UnplugWarning.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'
  import { toggleBootStatus, toggleAutoReboot } from '$lib/stores/websocket'

  import { URBIT_MODE } from '$lib/stores/data'

  export let patp
  export let ownShip
  export let running
  export let detectBootStatus
  export let detectAutoRestart
  export let tTogglePower

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  const dispatch = createEventDispatcher()

  function handleClick() {
    if ($URBIT_MODE) {
      openModal(FinalModal, {"component":"power","patp":patp})
    } else {
      dispatch("click")
    }
  }
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Power</div>
    <div class="check-wrapper">
      <div class="checkbox" on:click={()=>toggleBootStatus(patp)}>
      {#if detectBootStatus}
        <img class="checkmark" src={pfx+"/checkmark-white.svg"} alt="checkmark"/>
      {/if}
      </div>
      <div class="check-text" on:click={()=>toggleBootStatus(patp)}>Remember boot status after restart</div>
      <!--
      <div class="what">?</div>
      -->
    </div>
    <div class="check-wrapper">
      <div class="checkbox" on:click={()=>toggleAutoReboot(patp)}>
      {#if detectAutoRestart}
        <img class="checkmark" src={pfx+"/checkmark-white.svg"} alt="checkmark"/>
      {/if}
      </div>
      <div class="check-text" on:click={()=>toggleAutoReboot(patp)}>Reboot ship after crash</div>
    </div>
  </div>
  <div class="section-right">
    <UnplugWarning component={"power"} {ownShip}>
      <ToggleButton
        on:click={handleClick}
        on={running}
        loading={tTogglePower}
        />
    </UnplugWarning>
  </div>
</div>

<style>
  .check-wrapper {
    margin: 12px 0 0 8px;
    display: flex;  
    align-items: center;
    gap: 8px;
  }
  .checkbox {
    width: 24px;
    height: 24px;
    border: solid 1px var(--text-card-color);
    border-radius: 4px;
    cursor: pointer;
  }
  .checkmark {
    width: 16px;
    height: 16px;
    padding: 4px;
    cursor: pointer;
  }
  .check-text {
    font-size: 12px;
    color: var(--text-card-color);
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 150% */
    letter-spacing: -0.96px;
    cursor: pointer;
  }
  .what {
    margin: 0 8px;
    width: 20px;
    height: 20px;
    text-align: center;
    border: 1px solid #FFF;
    border-radius: 50%;
    cursor: pointer;
  }
  .what:hover {
    opacity: .2;
  }
</style>
