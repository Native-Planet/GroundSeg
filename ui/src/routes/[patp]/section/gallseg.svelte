<script>
  import ToggleButton from '$lib/togglebutton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  export let gallseg
  export let tGallsegInstalling = ""

  const dispatch = createEventDispatcher()
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">GroundSeg On Urbit</div>
    <div class="section-description">This allows you to manage GroundSeg from your Urbit ship</div>
  </div>
  <div class="section-right">
    <ToggleButton
      on:click={()=>dispatch("click")}
      loading={tGallsegInstalling.length > 0}
      on={gallseg}
      />
  </div>
</div>
