<script>
  import Fa from 'svelte-fa'
  import { faArrowRight } from '@fortawesome/free-solid-svg-icons';
  import { startramToggle, structure } from '$lib/stores/websocket'
  import ToggleButton from '$lib/ToggleButton.svelte'
  import StarTramInfo from './StarTramInfo.svelte'

  $: info = ($structure?.profile?.startram?.info) || {}
  $: renew = (info?.renew) || false
  $: expiry = (info?.expiry) || ""
  $: running = (info?.running) || false
  $: transition = ($structure?.profile?.startram?.transition) || {}
  $: tToggle = (transition?.toggle) || null


  /*
  $: registered = (info?.registered) || false
  $: region = (info?.region) || ""
  */
  /*
  $: endpoint = (info?.endpoint) || ""

  */
</script>

<div class="container">
  <div class="top">
    <StarTramInfo {renew} {expiry} />
    <div class="spacer"></div>
    <button class="new-key">New Key</button>
    <ToggleButton on:click={startramToggle} loading={tToggle} on={running} />
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
