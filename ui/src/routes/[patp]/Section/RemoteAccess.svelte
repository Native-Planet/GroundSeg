<script>
  import { openModal } from 'svelte-modals'
  import FinalModal from './FinalModal.svelte';
  import ToggleButton from '$lib/ToggleButton.svelte'
  import UnplugWarning from './UnplugWarning.svelte';
  import { toggleBackups } from '$lib/stores/websocket';
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  import Fa from 'svelte-fa'
  import { faPlugCircleExclamation } from '@fortawesome/free-solid-svg-icons';

  import { structure } from '$lib/stores/data'
  import { URBIT_MODE } from '$lib/stores/data'

  export let patp

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""
  $: ship = ($structure?.urbits?.[patp]) || {}
  $: wgRunning = ($structure?.profile?.startram?.info?.running) || false
  $: enableBackups = ship?.info?.enableBackups == undefined || ship?.info?.enableBackups // true by default

  export let remote
  export let remoteReady
  export let tToggleNetwork = ""
  export let ownShip
  let loading = false
  let awaitChange = false
  let lastState = enableBackups

  const dispatch = createEventDispatcher()

  function handleClick() {
    if ($URBIT_MODE) {
      openModal(FinalModal, {"component":"remote","patp":patp})
    } else {
      dispatch("click")
    }
  }

  const handleToggleBackups = () => {
    loading = true
    awaitChange = true
    lastState = enableBackups
    toggleBackups(patp, !enableBackups)
  }
</script>

<div class="section" class:disabled={!wgRunning || !remoteReady}>
  <div class="section-left">
    <div class="section-title">Remote Access</div>
    <div class="section-description">Access your ship via a StarTram connection</div>
      {#if remote}
        <div class="check-wrapper" class:disabled={loading}>
          <div class="checkbox" on:click={handleToggleBackups}>
            {#if enableBackups}
              <img class="checkmark" src={pfx+"/checkmark-white.svg"} alt="checkmark"/>
            {/if}
          </div>
          <div class="check-text" on:click={handleToggleBackups}>Daily remote encrypted chat backups</div>
        </div>
      {/if}
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
  .check-wrapper {
    margin: 32px 0 0 8px;
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
</style>
