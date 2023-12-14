<script>
  import ToggleButton from '$lib/togglebutton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  export let devMode
  export let tToggleDevMode = ""

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
      loading={tToggleDevMode.length > 0}
      on={devMode}
      />
  </div>
</div>
