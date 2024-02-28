<script>
  import { toggleExperimentalPenpai } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import ToggleButton from '$lib/ToggleButton.svelte'

  $: penpai = ($structure?.apps?.penpai?.info) || {}
  $: penpaiAllowed = (penpai?.allowed) || false
</script>

<div class="container">
  <div class="sys-title">PENPAI (EXPERIMENTAL)</div>
  <div class="wifi-toggle">
    <div class="wifi-text">Enable Feature</div>
    <ToggleButton
      on:click={toggleExperimentalPenpai}
      on={penpaiAllowed}
      />
  </div>
</div>

<style>
  .container {
    margin: 0;
  }
  .sys-title {
    margin-bottom: 56px;
  }
  .wifi-toggle {
    display: flex;
    align-items: center;
  }
  .wifi-text {
    flex: 1;
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
</style>
