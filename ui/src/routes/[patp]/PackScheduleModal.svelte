<script>
  //import Select from 'svelte-select';
  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'
  import Selector from './Selector.svelte'
  import Clock from './Clock.svelte'
  export let isOpen
 
  $: num = num > 1 ? num : 1
  let selectedOption = "Week"
  let selectedDay = "Wednesday"
  let selectedDate = 1
  let days = ["Monday","Tuesday","Wednesday","Thursday","Friday","Saturday","Sunday"]
  let dates = [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31]

  const handleClockChange = e => {
    console.log(e.detail)
  }
</script>

<Modal width={720}>
  {#if isOpen}
    <div class="wrapper">
      <div class="header">Schedule Pack</div>
      <div class="information">
        <div class="pack">
          Last: 1/3/2023 3:00 PM (3 days ago)
        </div>
        <div class="pack">
          Next: 5/3/2023 3:00 PM (In 4 days)
        </div>
      </div>

      <div class="macro">
        <div>Every</div>
        <input type="number" bind:value={num}/>
        <Selector {num} on:change={e=>selectedOption=e.detail}/>
      </div>

      <div class="micro">
        <div class="time-wrapper">
          <div class="micro-title">Time</div>
          <Clock on:select={handleClockChange} />
        </div>

        {#if selectedOption == "Week"}
          <div class="select-wrapper">
            <div class="micro-title">Day</div>
            <div class="day-wrapper">
              {#each days as d}
                <div
                  class="day"
                  on:click={()=>selectedDay=d}
                  class:active={d==selectedDay}>
                  {d}
                </div>
              {/each}
            </div>
          </div>
        {/if}

        {#if selectedOption == "Month"}
          <div class="select-wrapper">
            <div class="micro-title">Date</div>
            <div class="date-wrapper">
              {#each dates as n}
                <div
                  class="date"
                  on:click={()=>selectedDate=n}
                  class:active={n==selectedDate}>
                  {n}
                </div>
              {/each}
            </div>
          </div>
        {/if}
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
  .information {
    display: flex;
    gap: 32px;
  }
  .pack {
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 500;
    letter-spacing: -1.44px;
    line-height: 55px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    border: none;
    padding: 0 24px;
  }
  .macro {
    display: flex;
    gap: 16px;
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
    align-items: center;
  }
  .micro {
    display: flex;
    gap: 24px;
  }
  .time-wrapper {
    flex: 1;
  }
  .micro-title {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
    text-align: center;
  }
  input {
    width: 40px;
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    max-width: 460px;
    line-height: 55px;
    min-width: 55px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    border: none;
  }
  /* Hide spinners in number input for Webkit browsers */
  input[type="number"]::-webkit-inner-spin-button,
  input[type="number"]::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }
  /* Hide spinners in number input for Firefox */
  input[type="number"] {
    -moz-appearance: textfield;
  }
  .select-wrapper {
    flex: 1;
  }
  .day-wrapper {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: center;
  }
  .date-wrapper {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .day {
    user-select: none;
    flex-basis: 45%;
    cursor: pointer;
    padding: 16px 0;
    text-align: center;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
  }
  .date {
    flex-basis: 11%;
    padding: 4px 0;
    text-align: center;
    user-select: none;
    cursor: pointer;
    border: solid 1px var(--btn-secondary);
    border-radius: 4px;
  }
  .active {
    background: var(--btn-secondary);
    color: var(--text-card-color);
  }
  .button-wrapper {
    margin-top: 64px;
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
