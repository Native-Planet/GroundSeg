<script>
  import ToggleButton from '../ToggleButton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'

  export let running
  export let tTogglePower

  const dispatch = createEventDispatcher()
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Power</div>
    <div class="check-wrapper">
      <div class="checkbox"></div>
      <div class="check-text">Autoboot after restart</div>
    </div>
  </div>
  <div class="section-right">
    <ToggleButton
      on:click={()=>dispatch("click")}
      on={running}
      loading={tTogglePower}
      />
  </div>

  <!-- Urbit Ship Status 
  <div class="section">
    <div class="section-left">
      <div class="section-title">Remember Urbit Boot Status</div>
      <div class="section-description">This enables your ship to autoboot after a device restart</div>
    </div>
    <div class="section-right">
      <div class="checkbox" class:highlight={detectBootStatus} on:click={()=>toggleAutoBoot(patp)}>
        {#if detectBootStatus}
          <div class="icon"><Fa icon={faCheck} size="1x"/></div>
        {/if}
      </div>
    </div>
  </div>
  -->
</div>

<style>
  .check-wrapper {
    margin: 8px 0 0 4px;
    display: flex;  
    gap: 8px;
  }
  .checkbox {
    width: 12px;
    height: 12px;
    border: solid 1px var(--text-card-color);
    border-radius: 4px;
  }
  .check-text {
    font-size: 12px;
  }
</style>
