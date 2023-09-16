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
  import StarTramModify from './StarTramModify.svelte'

  // Info
  $: info = ($structure?.profile?.startram?.info) || {}
  $: renew = (info?.renew) || false
  $: expiry = (info?.expiry) || ""
  $: running = (info?.running) || false
  $: registered = (info?.registered) || false

  // Transition
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tToggle = (transition?.toggle) || null
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
  <StarTramModify />
</div>

<style>
  .container {
    margin: auto;
    width: calc(1104px - (56px * 2));
    max-width: 98vw;
    padding: 56px;
  }
  .top {
    display: flex;
    gap: 24px;
  }
  .new-key {
    background: var(--btn-secondary);
    height: 65px;
    padding: 0 48px;
    border-radius: 16px;
    font-size: 24px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .spacer {
    flex: 1;
  }
</style>
