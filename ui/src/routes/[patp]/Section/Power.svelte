<script>
  import ToggleButton from '$lib/ToggleButton.svelte'
  // Style
  import "../theme.css"
  import { createEventDispatcher } from 'svelte'
  import { toggleBootStatus } from '$lib/stores/websocket'

  export let patp
  export let running
  export let detectBootStatus
  export let tTogglePower

  const dispatch = createEventDispatcher()
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Power</div>
    <div class="check-wrapper">
      <div class="checkbox" on:click={()=>toggleBootStatus(patp)}>
      {#if detectBootStatus}
        <img class="checkmark" src="/checkmark-white.svg" alt="checkmark"/>
      {/if}
      </div>
      <div class="check-text" on:click={()=>toggleBootStatus(patp)}>Remember boot status after restart</div>
      <!--
      <div class="what">?</div>
      -->
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
    cursor: pointer;
  }
  .checkmark {
    width: 16px;
    height: 16px;
    padding: 4px;
    cursor: pointer;
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
    cursor: pointer;
  }
  .what {
    margin: 0 8px;
    width: 20px;
    height: 20px;
    text-align: center;
    border: 1px solid #FFF;
    border-radius: 50%;
    cursor: pointer;
  }
  .what:hover {
    opacity: .2;
  }
</style>
