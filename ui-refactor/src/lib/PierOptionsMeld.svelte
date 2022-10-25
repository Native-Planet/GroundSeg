<script>
  import { onMount } from 'svelte'
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faCaretLeft, faCaretRight } from '@fortawesome/free-solid-svg-icons'
  import SveltyPicker from 'svelty-picker'
  //import { createEventDispatcher } from 'svelte'

  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let timeNow, frequency

  let exportButtonText = 'Export Urbit Pier',
    deleteButtonText = 'Delete Urbit Pier', 
    inView = true,
    meldTime = '00:00',
    cloneFreq

  onMount(()=> cloneFreq = frequency)

  //const dispatch = createEventDispatcher()

  const sendMeldPoke = () => {
    console.log("meldd")
  }

  const saveMeldChanges = () => {
    console.log("meld settings saved")
  }

</script>

  <div class="option-title">Schedule Meld</div>

  <!-- frequency selector -->
  <div class="day">
    <button disabled={cloneFreq <= 1} class="day-button" on:click={()=> cloneFreq = --cloneFreq }>
      <Fa icon={faCaretLeft} size="1x" />
    </button>

    <div class="day-text">Every</div>
    <input type="number" class="day-input" bind:value={cloneFreq} min=1 max=365 />
    <div class="day-text">days</div>

    <button class="day-button" on:click={()=>cloneFreq = ++cloneFreq}>
      <Fa icon={faCaretRight} size="1x" />
    </button>

  </div>

  <div class="day">
    <div class="day-text">at</div>

    <!-- temp -->

    <input id="hour" type="number" />
  </div>


  <!-- Current time on host device -->
  <div class="current-time">Current time: {timeNow.slice(5, -4)} UTC</div>


<!--
  <div class="meld-schedule">
    <!-- current time --

      <!-- frequency selector --
      <div class="day">
        <button disabled={cloneFreq <= 1} class="day-button" on:click={()=> cloneFreq = --cloneFreq }>
          <Fa icon={faCaretLeft} size="1x" />
        </button>
        <div class="day-text">Every</div>
        <input type="number" class="day-input" bind:value={cloneFreq} min=1 max=365 />
        <div class="day-text">days</div>
        <button class="day-button" on:click={()=>cloneFreq = ++cloneFreq}>
          <Fa icon={faCaretRight} size="1x" />
        </button>
      </div>

      <div class="current-time">at</div>

      <!-- set meld time 
        using this package is temporary solution. Need to write our own.
      --
      <div style="color:#fff;width:60px;margin:auto;overflow:hidden;border-radius:8px;">
        <SveltyPicker inputClasses="form-control" theme="clock-colors" format="hh:ii" bind:value={meldTime}></SveltyPicker>
      </div>
    </div>

  <!-- Meld Actions --
  <div class="admin-wrapper">
    <div class="option-title">Meld Actions</div>

    <!-- Save new meld schedule --
    <PrimaryButton
      noMargin={true}
      standard="{frequency == cloneFreq ? "No" : "Save"} changes"
      success="changes saved!"
      status={frequency == cloneFreq ? 'disabled' : 'standard'}
      on:click={saveMeldChanges}
      />

    <!-- Meld now --
    {#if running}
      <PrimaryButton
        background="#FFFFFF4D"
        noMargin={true}
        standard="Meld now"
        success="Meld poke successful. Please give it a moment"
        on:click={sendMeldPoke}
        />
    {/if}

  </div>

</div>

    -->
<style>
  :global(.clock-colors) {
    --sdt-primary: #008eff;
    --sdt-color: #fff;
    --sdt-bg-main: #040404BF;
    --sdt-bg-today: var(--sdt-primary);
    --sdt-btn-bg-hover: red;
    --sdt-btn-header-bg-hover: #dfdfdf;
    --sdt-clock-bg: #eeeded;
    --sdt-clock-bg-minute: rgb(238, 237, 237, 0.25);
  }
  .meld-schedule {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .option-title {
    font-size: 14px;
    color: inherit;
  }

  .current-time {
    font-size:12px;
  }
  .day {
    display: flex;
    gap: 2px;
    justify-content: center;
  }
  .day-text {
    font-size: 12px;
    color: inherit;
  }
  .day-button {
    color: inherit;
  }
  .day-input {
    text-align: center;
    color: inherit;
    background: none;
    border: none;
    width: 24px;
    font-size: 12px;
    font-family: inherit;
    padding: none;
    margin: none;
  }
  input:focus {outline: none;}
  input::-webkit-outer-spin-button,
  input::-webkit-inner-spin-button {-webkit-appearance: none;margin: 0;}
  input[type=number] {-moz-appearance: textfield;}

  .admin-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .export-pier {
    color: orange;
    cursor: pointer;
  }

  .delete-pier {
    color: red;
    cursor: pointer;
  }
</style>
