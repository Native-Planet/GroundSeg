<script>
  // Style
  import "../theme.css"
  import { openModal } from 'svelte-modals'
  import PackScheduleModal from '../PackScheduleModal.svelte'
  import { urthPackMeld, marsPack } from '$lib/stores/websocket'
  import { createEventDispatcher } from 'svelte'

  export let patp

  const dispatch = createEventDispatcher()


  const handleModal = () => {
    openModal(PackScheduleModal)
  }


</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Pack Pier</div>
    <div class="section-description">
      This function will refragement your ship's memory capacity, optimizing its performance. We recommend scheduling these once a week
    </div>
  </div>
  <div class="section-right">
    <div class="btn-wrapper">
      <div class="spacer"></div>
      <button class="start urth" on:click={()=>urthPackMeld(patp)}>Pack & Meld</button>
      <button class="start" on:click={()=>marsPack(patp)}>Pack</button>
      <button class="calendar" on:click={handleModal}>
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
