<script>
  // Style
  import "../theme.css"
  import { urbitChop } from '$lib/stores/websocket'
  import { createEventDispatcher } from 'svelte'

  export let patp

  const dispatch = createEventDispatcher()


</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Chop</div>
    <div class="section-description">
      Make pier smol
    </div>
  </div>
  <div class="section-right">
    <div class="btn-wrapper">
      <div class="spacer"></div>
      <button class="start" on:click={()=>urbitChop(patp)}>Chop</button>
      <button class="calendar">
        <img src="/calendar.svg" alt="calendar icon" width="20px" height="20px"/>
      </button>
    </div>
  </div>
</div>

<style>
  .btn-wrapper {
    display: flex; 
    gap: 8px;
  }
  .spacer {
    flex: 1;
  }
  button {
    cursor: pointer;
  }
  .start {
    display: flex;
    padding: 20px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    border-radius: 16px;
    background: #2C3A2E;
    color: #FFF;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 75% */
    letter-spacing: -1.92px;
  }
  .urth {
    background: var(--text-color);  
  }
  .calendar {
    padding: 20px 22px;
    background: #313933;
    color: #FFF;
    border-radius: 16px;
  }
</style>
