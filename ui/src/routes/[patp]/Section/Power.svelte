<script>
  import ToggleButton from '$lib/ToggleButton.svelte'
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
</div>

<style>
  .check-wrapper {
    margin: 12px 0 0 8px;
    display: flex;  
    align-items: center;
    gap: 8px;
  }
  .checkbox {
    width: 24px;
    height: 24px;
    border: solid 1px var(--text-card-color);
    border-radius: 4px;
  }
  .check-text {
    font-size: 12px;
    color: var(--text-card-color);
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 150% */
    letter-spacing: -0.96px;
  }
</style>
