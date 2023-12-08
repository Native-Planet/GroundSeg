<script>
  import ToggleButton from '$lib/ToggleButton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'
  import { structure } from '$lib/stores/data'

  $: wgRunning = ($structure?.profile?.startram?.info?.running) || false

  export let remote
  export let remoteReady
  export let tToggleNetwork = ""

  const dispatch = createEventDispatcher()
</script>

<div class="section" class:disabled={!wgRunning || !remoteReady}>
  <div class="section-left">
    <div class="section-title">Remote Access</div>
    <div class="section-description">Access your ship via a StarTram connection</div>
  </div>
  <div class="section-right">
    <ToggleButton
      on:click={()=>dispatch("click")}
      on={remote}
      loading={tToggleNetwork.length > 0}
      />
  </div>
</div>

<style>
  .disabled {
    opacity: .4;
    pointer-events: none;
  }
</style>
