<script>
  import ToggleButton from '$lib/ToggleButton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  export let devMode

  const dispatch = createEventDispatcher()
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Developer Mode</div>
    <div class="section-description">This enables your ship to be able to debug from another computer</div>
  </div>
  <div class="section-right">
    <ToggleButton
      on:click={()=>dispatch("click")}
      on={devMode}
      />
  </div>
</div>
