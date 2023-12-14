<script>
  import { createEventDispatcher } from 'svelte'
  import { structure } from '$lib/stores/data'
  const dispatch = createEventDispatcher()
  export let patp

  $: packTime = ($structure?.urbits?.[patp]?.info?.packTime)

  $: hourUnadjusted = packTime.slice(0,2)
  $: hourInt = Number(hourUnadjusted)
  $: hourStr = hourInt > 11 ? (hourInt - 12).toString() : hourInt.toString()
  $: hour = hourStr.length == 1 ? "0" + hourStr : hourStr

  $: minute = packTime.slice(2,4)

  $: meridian = hourInt > 11 ? "PM" : "AM"

  let hours = ['01', '02', '03', '04', '05', '06', '07', '08', '09', '10', '11', '12']
  let minutes = [
    '00', '01', '02', '03', '04', '05', '06', '07', '08', '09', '10', '11', 
    '12', '13', '14', '15', '16', '17', '18', '19', '20', '21', '22', '23',
    '24', '25', '26', '27', '28', '29', '30', '31', '32', '33', '34', '35',
    '36', '37', '38', '39', '40', '41', '42', '43', '44', '45', '46', '47',
    '48', '49', '50', '51', '52', '53', '54', '55', '56', '57', '58', '59'
  ]

  let hourMenu = false
  let minuteMenu = false

  $: time = dispatch("select",handleTime(hour,minute,meridian))

  const handleTime = (h,m,mer) => {
    const intHour = parseInt(h, 10)
    if (mer == "AM") {
      if (intHour > 11) {
        return "00" + minute
      }
      return hour + minute
    }
    if (intHour == 12) {
      return hour + minute
    }
    return (intHour + 12) + minute
  }

  const toggleHourMenu = () => {
    hourMenu = !hourMenu
    minuteMenu = false
  }

  const toggleMinuteMenu = () => {
    minuteMenu = !minuteMenu
    hourMenu = false
  }

  const handleHourSelect = h => {
    hour = h
    hourMenu = false
  }

  const handleMinuteSelect = m => {
    minute = m
    minuteMenu = false
  }
</script>

<div class="wrapper">
  <div class="main-spacer"></div>
  <div class="selector-wrapper">
    <div class="selector" on:click={toggleHourMenu}>{hour}</div>
    {#if hourMenu}
      <div class="options">
        {#each hours as h}
          <div
            class="option"
            on:click={()=>handleHourSelect(h)}
            class:active={h==hour}>
            {h}
          </div>
        {/each}
      </div>
    {/if}
  </div>
  <div class="spacer">:</div>
  <div class="selector-wrapper">
    <div class="selector" on:click={toggleMinuteMenu}>{minute}</div>
    {#if minuteMenu}
      <div class="options">
        {#each minutes as m}
          <div
            class="option"
            on:click={()=>handleMinuteSelect(m)}
            class:active={m==minute}>
            {m}
          </div>
        {/each}
      </div>
    {/if}
  </div>
  <div class="am-pm">
    <div class="item" on:click={()=>meridian="AM"} class:active={meridian=="AM"}>AM</div>
    <div class="item" on:click={()=>meridian="PM"} class:active={meridian=="PM"}>PM</div>
  </div>
  <div class="main-spacer"></div>
</div>

<style>
  .wrapper {
    display: flex;
    gap: 16px;
    margin-bottom: 128px;
  }
  .main-spacer {
    flex: 1;
  }
  .selector-wrapper {
    position: relative;
  }
  .selector {
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    line-height: 55px;
    width: 80px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    border: none;
  }
  .options {
    position: absolute;
    top: 62px;
    background: var(--Gray-100, #DDE3DF);
    height: 160px;
    overflow-y: scroll;
    width: 100%;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
  }
  /* Hide scrollbar for Chrome, Safari and Opera */
  .options::-webkit-scrollbar {
    display: none;
  }

  /* Hide scrollbar for Firefox */
  .options {
    scrollbar-width: none; /* Firefox 64+ */
  }

  /* Hide scrollbar for IE and Edge */
  .options {
    -ms-overflow-style: none; /* IE 11+ */
  }
  .option {
    padding: 4px 0;
    cursor: pointer;
  }
  .option:hover {
    font-weight: 500;
  }
  .spacer {
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    line-height: 55px;
  }
  .am-pm {
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    display: flex;
    gap: 4px;
    flex-direction: column;
    height: 55px;
    width: 80px;
  }
  .item {
    flex: 1;
    border: solid 1px var(--btn-secondary);
    border-radius: 8px;
    line-height: 24px;
    cursor: pointer;
  }
  .active {
    background: var(--btn-secondary);
    color: var(--text-card-color);
  }
</style>
