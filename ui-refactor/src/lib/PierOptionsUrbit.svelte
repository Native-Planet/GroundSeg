<script>
  import { onMount, onDestroy } from 'svelte'

  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faCaretLeft, faCaretRight } from '@fortawesome/free-solid-svg-icons'
  import SveltyPicker from 'svelty-picker'
  //import { createEventDispatcher } from 'svelte'

  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, running

  let exportButtonText = 'Export Urbit Pier',
    deleteButtonText = 'Delete Urbit Pier',
    frequency = 7, now, inView = true,
    meldTime = '00:00'

  //const dispatch = createEventDispatcher()

  const exportUrbitPier = () => {
    console.log("exported pier")
  }
 
  const deleteUrbitPier = () => {
    console.log("deleted pier") 
  }

  const sendMeldPoke = () => {
    console.log("meldd")
  }

  const saveMeldChanges = () => {
    console.log("meld settings saved")
  }

  const currentTime = () => {
    if (inView) {
      let time = Date.now()
      let dateObj = new Date(time)
      now = dateObj.toUTCString().slice(17, -4)
      setTimeout(currentTime, 1000)
    }
  }

  onMount(()=> currentTime())
  onDestroy(()=> inView = !inView)

</script>

<div class="main-wrapper">

  <!-- Admin options -->
  <div class="admin-wrapper">
    <div class="option-title">Admin Actions</div>
    <button class="export-pier" on:click={exportUrbitPier}>{exportButtonText}</button>
    <button class="delete-pier" on:click={deleteUrbitPier}>{deleteButtonText}</button>
  </div>

  <!-- Meld scheduling -->
  <div class="meld-wrapper">
    <div class="meld-schedule">
      <div class="option-title">Schedule Meld</div>

      <!-- current time -->
      <div class="current-time">Current time: {now} UTC</div>

      <!-- frequency selector -->
      <div class="day">
        <button disabled={frequency <= 1} class="day-button" on:click={()=>frequency = --frequency }>
          <Fa icon={faCaretLeft} size="1x" />
        </button>
        <div class="day-text">Frequency (days):</div>
        <input type="number" class="day-input" bind:value={frequency} min=1 max=365 />
        <button class="day-button" on:click={()=>frequency = ++frequency}>
          <Fa icon={faCaretRight} size="1x" />
        </button>
      </div>

      <!-- set meld time -->
      <div style="color:#fff;width:60px;margin:auto;overflow:hidden;border-radius:8px;">
        <SveltyPicker inputClasses="form-control" theme="clock-colors" format="hh:ii" bind:value={meldTime}></SveltyPicker>
      </div>
    </div>

  </div>

  <!-- Meld Actions -->
  <div class="admin-wrapper">
    <div class="option-title">Meld Actions</div>

    <!-- Save new meld schedule -->
    <PrimaryButton
      noMargin={true}
      standard="Save changes"
      success="changes saved!"
      status="disabled"
      on:click={saveMeldChanges}
      />

    <!-- Meld now -->
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
  .main-wrapper {
    display: flex;
    text-align: center;
    padding: 20px;
    align-items: start;
  }

  .meld-wrapper {
    flex:1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .meld-schedule {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .option-title {
    font-size: 14px;
    color: inherit;
    margin-bottom: 12px;
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
