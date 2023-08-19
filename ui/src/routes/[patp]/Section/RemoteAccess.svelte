<script>
  import ToggleButton from '../ToggleButton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  export let remote

  const dispatch = createEventDispatcher()
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Remote Access</div>
    <div class="section-description">Access your ship via a StarTram connection</div>
  </div>
  <div class="section-right">
    <ToggleButton
      on:click={()=>dispatch("click")}
      on={remote}
      />
  </div>
</div>
