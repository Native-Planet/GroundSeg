<script>
  import { onMount } from 'svelte'
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faCaretLeft, faCaretRight } from '@fortawesome/free-solid-svg-icons'
  import SveltyPicker from 'svelty-picker'
  import { Listbox, ListboxButton, ListboxOptions, ListboxOption } from "@rgossiaux/svelte-headlessui"

  //import { createEventDispatcher } from 'svelte'

  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let timeNow, frequency, running, name, meldHour, meldMinute
    
  let selectedHour = meldHour, selectedMinute = meldMinute, meldSetStatus = 'standard'

  let exportButtonText = 'Export Urbit Pier', deleteButtonText = 'Delete Urbit Pier'
  let inView = true
  let cloneFreq
  let minutes = Array.from(Array(60).keys()) 
  let hours = Array.from(Array(24).keys())

  onMount(()=> cloneFreq = frequency)

  //const dispatch = createEventDispatcher()

  const sendMeldPoke = () => {
    console.log("meldd")
  }

  const saveMeldChanges = () => {
		fetch($api + '/urbit?urbit_id=' + name, {
		method: 'POST',
		headers: {'Content-Type': 'application/json'},
		body: JSON.stringify({'app':'pier','data':'schedule-meld','frequency':cloneFreq,'hour':selectedHour,'minute':selectedMinute})
		})
      .then(d=>d.json())
      .then(r=>{
        if (r === 200) {
          console.log(r)
          meldSetStatus = 'success'
          setTimeout(()=>{
            meldSetStatus = 'standard'
          }, 3000)}
        if (r === 400) {
          meldSetStatus = 'failure'
          setTimeout(()=>meldSetStatus = 'standard', 3000)
      }})
		  .catch(err => console.log(err))
    }

</script>

<div class="option-title">Schedule Meld</div>

<div class="panel">
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

    <!-- hour selector -->
    <Listbox class="time-box" value={selectedHour} on:change={(e) => (selectedHour = e.detail)}>
      <ListboxButton class="time-selector">{selectedHour < 10 ? "0" : ""}{selectedHour}</ListboxButton>
      <ListboxOptions class="time-list">
        {#each hours as hour}
          <ListboxOption class="time-option" value={hour}>
            {hour < 10 ? "0" : ""}{hour}
          </ListboxOption>
        {/each}
      </ListboxOptions>
    </Listbox>

    <div class="day-text">:</div>

    <!-- minute selector -->
    <Listbox value={selectedMinute} on:change={(e) => (selectedMinute = e.detail)}>
      <ListboxButton class="time-selector">{selectedMinute < 10 ? "0" : ""}{selectedMinute}</ListboxButton>
      <ListboxOptions class="time-list">
        {#each minutes as minute}
          <ListboxOption value={minute}>
            {minute < 10 ? "0" : ""}{minute}
          </ListboxOption>
        {/each}
      </ListboxOptions>
    </Listbox>

  </div>

  <!-- Current time on host device -->
  <div class="day">
    <div class="current-time">Current time: {timeNow.slice(5, -4)} UTC</div>
  </div>

  <div class="day-action">
  <!-- Save new meld schedule -->
  <PrimaryButton
    noMargin={true}
    standard="{
      frequency != cloneFreq 
      || selectedHour != meldHour 
      || selectedMinute != meldMinute
      ? "Save" : "No"
    } changes"
    success="changes saved!"
    failure="failed to set meld schedule"
    status={
      frequency != cloneFreq 
      || selectedHour != meldHour 
      || selectedMinute != meldMinute
      ? meldSetStatus : 'disabled'
    }
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
	:global(.time-list::-webkit-scrollbar) {display: none;}
  :global(.time-selector) {
    padding: none;
    font-size: 12px;
    font-family: inherit;
    color: inherit;
    background: #FFFFFF4D;
    border-radius: 4px;
    position: relative;
  }
  :global(.time-list) {
    font-size: 12px;
    text-align: center;
    background: #040404;
    position: absolute;
    padding: 0 6px 0 6px;
    max-height: 64px;
    -ms-overflow-style: none;
		scrollbar-width: none;
    overflow: scroll;
    list-style-type: none;
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
    gap: 4px;
    justify-content: center;
    align-items: center;
    padding: 4px;
  }
  .day-action {
    display: flex;
    gap: 6px;
    justify-content: center;
    align-items: center;
    padding: 4px;
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
</style>
