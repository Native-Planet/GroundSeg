<script>
  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'
  export let isOpen
 
  $: num = num > 1 ? num : 1
  let days = ["monday","tuesday","wednesday","thursday","friday","saturday","sunday"]
</script>

<Modal width={800}>
  {#if isOpen}
    <div class="wrapper">
      <div class="header">Schedule Pack</div>
      <div class="information">
        last pack: somedatetime
        next pack: somedatetime/stopped
      </div>

      <div class="macro">
        <div>Every</div>
        <input type="number" bind:value={num}/>
        <div>Days (sel)</div>
      </div>

      <div class="micro">
        <!-- days, weeks, months -->
        <div class="time">
          <div>02:45 AM</div>
        </div>
        <div class="day-wrapper">
          {#each days as d}
            <div class="day">{d}</div>
          {/each}
        </div>
        <div class="date">date selector (months)</div>
      </div>

      <div class="button-wrapper">
        <button>Save Schedule</button>
        <div class="spacer"></div>
        <button class="stop">Pause Schedule</button>
      </div>
    </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .header {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .macro {
    display: flex;
    gap: 48px;
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    max-width: 460px;
    margin: 32px 0;
  }
  .day-wrapper {
    display: flex;
    gap: 32px;
  }
  .day {
  }
  .button-wrapper {
    margin-top: 32px;
    display: flex;
  }
  .spacer {
    flex: 1;
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: var(--btn-primary);
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    cursor: pointer;
    height: 65px;
  }
  button:disabled {
    opacity: .6;
    pointer-events:none;
  }
  .stop {
    background: var(--btn-secondary);
  }
</style>
