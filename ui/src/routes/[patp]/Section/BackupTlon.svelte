<script>
  import { openModal } from 'svelte-modals'
  import TlonBackupModal from '../TlonBackupModal.svelte'
  import ToggleButton from '$lib/ToggleButton.svelte'
  import UnplugWarning from './UnplugWarning.svelte';
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  import Fa from 'svelte-fa'
  import { faGear } from '@fortawesome/free-solid-svg-icons';

  import { structure } from '$lib/stores/data'
  import { URBIT_MODE } from '$lib/stores/data'

  $: wgRunning = ($structure?.profile?.startram?.info?.running) || false
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  export let patp
  export let remoteReady
  export let localTlonBackupsEnabled
  export let remoteTlonBackupsEnabled

  export let tLocalTlonBackupsEnabled
  export let tRemoteTlonBackupsEnabled

  const dispatch = createEventDispatcher()

  const handleModal = () => {
    openModal(TlonBackupModal,{"patp":patp})
  }
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Encrypted Chat Backups</div>
    <div class="section-description">Daily local Tlon chat backups</div>
    <div class="check-wrapper" class:disabled={!wgRunning || !remoteReady}>
      <div class="checkbox" on:click={()=>dispatch("remote")}>
        {#if remoteTlonBackupsEnabled}
          <img class="checkmark" src={pfx+"/checkmark-white.svg"} alt="checkmark"/>
        {/if}
      </div>
      <div class="check-text" on:click={()=>dispatch("remote")}>Automatic upload backups to StarTram</div>
    </div>
  </div>
  <div class="section-right">
    <div class="btn-wrapper">
      <div class="spacer"></div>
      <button class="calendar" on:click={handleModal}>
        <Fa icon={faGear} size="1.5x" />
      </button>
      <ToggleButton
        on:click={()=>dispatch("local")}
        on={localTlonBackupsEnabled}
        loading={tLocalTlonBackupsEnabled.length > 0}
        />
    </div>
  </div>
</div>

<style>
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
  .disabled {
    opacity: .4;
    pointer-events: none;
  }
  .btn-wrapper {
    display: flex; 
    gap: 8px;
  }
  .spacer {
    flex: 1;
  }
  .calendar {
    width: 65px;
    background: #313933;
    color: #FFF;
    border-radius: 16px;
  }
</style>
