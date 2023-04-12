<script>
  import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'
  import { api } from '$lib/api'
  import Fa from 'svelte-fa'
  import { faCaretLeft, faCaretRight } from '@fortawesome/free-solid-svg-icons'
  import { faCircleQuestion} from '@fortawesome/free-regular-svg-icons'

  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import TimeSelector from '$lib/TimeSelector.svelte'

  export let timeNow
  export let disabled
  export let frequency
  export let running
  export let name
  export let meldHour
  export let meldMinute
  export let meldOn
  export let meldLast
  export let meldNext
    
  let showInfo = false

  let selectedHour = meldHour, selectedMinute = meldMinute, meldSetStatus = 'standard', meldNowStatus = 'standard'

  let exportButtonText = 'Export Urbit Pier', deleteButtonText = 'Delete Urbit Pier'
  let inView = true
  let cloneFreq
  let minutes = Array.from(Array(60).keys()) 
  let hours = Array.from(Array(24).keys())

  onMount(()=> cloneFreq = frequency)

  const sendMeldPoke = () => {
    meldNowStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
		method: 'POST',
        credentials: "include",
		headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({'app':'pier','data':'do-meld'})
		})
      .then(d=>d.json())
      .then(r=> {
        console.log(r)
        meldNowStatus = 'success'
        setTimeout(()=>{meldNowStatus = 'standard'}, 3000)
      })
		  .catch(err => console.log(err))
  }

  const toggleMeldSchedule = () => {
		fetch($api + '/urbit?urbit_id=' + name, {
		method: 'POST',
    credentials: "include",
		headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({'app':'pier','data':'toggle-meld'})
		})
      .then(d=>d.json())
      .then(r=>console.log(r))
		  .catch(err => console.log(err))
  }

  const saveMeldChanges = () => {
		fetch($api + '/urbit?urbit_id=' + name, {
		method: 'POST',
    credentials: "include",
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

<div class="bg" class:disabled={disabled}>
  <div class="title-wrapper">
    <div class="title-text">Schedule Pack & Meld</div>
    <div class="question-mark" on:click={()=>showInfo = !showInfo}><Fa icon={faCircleQuestion} size="1x" /></div>
    <div in:scale={{duration:100,delay:300, amount:10}} on:click={toggleMeldSchedule} class="switch-wrapper">
      <div class="switch {meldOn ? "on" : "off"}"></div>
    </div>
  </div>

  {#if showInfo}
    <div class="title-info">Defragment and deduplicate your memory. Helps improve performance!</div>
  {/if}

  <div class="panel">

    <!-- frequency selector -->
    <div class="day">
      <button disabled={cloneFreq <= 1} class="day-button" on:click={()=> cloneFreq = --cloneFreq }>
        <Fa icon={faCaretLeft} size="1x" />
      </button>

      <div class="day-text">Every</div>
      <input type="number" class="day-input" bind:value={cloneFreq} min=1 max=365 />
      <div class="day-text">day{cloneFreq > 1 ? "s" : ""}</div>

      <button class="day-button" on:click={()=>cloneFreq = ++cloneFreq}>
        <Fa icon={faCaretRight} size="1x" />
      </button>

    </div>

    <div class="day">

      <div class="day-text">at</div>

      <!-- hour selector -->
      <TimeSelector
        value={selectedHour}
        listOptions={hours}
        on:change={e => selectedHour = e.detail} 
      />

      <div class="day-text">:</div>

      <!-- minute selector -->
      <TimeSelector
        value={selectedMinute}
        listOptions={minutes}
        on:change={e => selectedMinute = e.detail} 
      />

    </div>

    <!-- Current time on host device -->
    <div class="day">
      <div class="current-time">Current time: {timeNow.slice(5, -4)} UTC</div>
    </div>

    <div class="day">
      <div class="current-time">Last meld: {meldLast.slice(12,-13) < 1971 ? "Never" : meldLast.slice(5, -4) + " UTC"}</div>
    </div>

    {#if meldOn}
    <div class="day">
      <div class="current-time">Next: {meldNext.slice(5, -4)} UTC</div>
    </div>
    {/if}

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
        status={meldNowStatus}
        loading="Attempting to send poke"
        noMargin={true}
        standard="Pack and Meld now"
        success="Meld poke sent, check your logs!"
        on:click={sendMeldPoke}
        />
    {/if}
    </div>
  </div>
</div>

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .title-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .question-mark {
    margin: 0 16px 0 8px;
    padding-top: 1px;
    cursor: pointer;
  }
  .title-text {
    float: left;
    font-size: 14px;
    color: inherit;
  }
  .title-info {
    font-size: 11px;
    margin: 8px 20px;
  }
  .switch-wrapper {
    margin-left: 6px;
    float: right;
    cursor: pointer;
    border-radius: 8px;
    width: 32px;
    height: 12px;
    background: #ffffff4d;
    padding: 2px;
  }
  .switch {
    height: 100%;
    width: 19px;
    border-radius: 6px;
  }
  .on {
    background: #008eff;
    float: right;
  }
  .off {
    background: #000;
    float: left;
    opacity: .2;
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
  .disabled {
    opacity: .6;
    pointer-events: none;
    background: #FF000033;
    color: #ffffff4d;
  }
  input:focus {outline: none;}
  input::-webkit-outer-spin-button,
  input::-webkit-inner-spin-button {-webkit-appearance: none;margin: 0;}
  input[type=number] {-moz-appearance: textfield;}
</style>
