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
  export let remoteReady
  export let toggleBackups
  export let tToggleBackups

  const dispatch = createEventDispatcher()
</script>

<div class="section" class:disabled={!wgRunning || !remoteReady}>
  <div class="section-left">
    <div class="section-title">Encrypted Remote Chat Backups</div>
    <div class="section-description">Daily Tlon chat backups to StarTram</div>
  </div>
  <div class="section-right">
    <ToggleButton
      on:click={()=>dispatch("click")}
      on={toggleBackups}
      loading={tToggleBackups.length > 0}
      />
  </div>
</div>

<style>
  .disabled {
    opacity: .4;
    pointer-events: none;
  }
</style>
