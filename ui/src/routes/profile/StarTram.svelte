<script>
  // Modals
  import { openModal } from 'svelte-modals'
  import RegisterModal from './RegisterModal.svelte'
  // Icons
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
  // Websocket
  import { startramToggle, structure } from '$lib/stores/websocket'
  // Components
  import ToggleButton from '$lib/ToggleButton.svelte'
  import StarTramInfo from './StarTramInfo.svelte'

  // Info
  $: info = ($structure?.profile?.startram?.info) || {}
  $: renew = (info?.renew) || false
  $: expiry = (info?.expiry) || ""
  $: running = (info?.running) || false
  $: registered = (info?.registered) || false

  // Transition
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tToggle = (transition?.toggle) || null
  //$: region = (info?.region) || "us-east"
  //$: endpoint = (info?.endpoint) || ""
</script>

<div class="container">
  <div class="top">
    <StarTramInfo {renew} {expiry} {registered} />
    <div class="spacer"></div>
    <button
      on:click={()=>openModal(RegisterModal)}
      class="new-key">
      New Key
    </button>
    {#if registered}
      <ToggleButton on:click={startramToggle} loading={tToggle} on={running} />
    {/if}
  </div>
</div>

<style>
  .container {
    margin: auto;
    width: calc(1104px - 80px);
    max-width: 98vw;
    padding: 40px;
  }
  .top {
    display: flex;
    gap: 24px;
  }
  .new-key {
    background-color: var(--btn-secondary);
    border-radius: 12px;
    color: var(--text-card-color);
    height: 42px;
    padding: 0 48px;
    font-family: var(--regular-font);
    font-size: 12px;
    cursor: pointer;
  }
  .spacer {
    flex: 1;
  }
</style>
