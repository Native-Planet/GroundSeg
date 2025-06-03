<script>
  import { openModal } from 'svelte-modals'
  import { snapTimeChangeActive } from '../store'
  import SnapTime from '../SnapTime.svelte'

  import { onMount, afterUpdate } from 'svelte'

  export let patp
  export let min = 10
  export let max = 90
  export let snapTime

  let slide
  let curSnapTime = snapTime

  let active = false
  $: range = max - min + 1
  $: step = Math.ceil((max - min) / 10)

  onMount(()=> {
    slide.addEventListener("mouseup", handleMouseUp);
  })

  let readyToClose = false
  afterUpdate(()=> {
    if ($snapTimeChangeActive) {
      readyToClose = true
    }
    if (readyToClose) {
      if (!$snapTimeChangeActive) {
         
      }
    }
  })

  const handleMouseUp = () => {
    if (curSnapTime != snapTime) {
      openModal(SnapTime,{"patp":patp,"snapTime":snapTime,"curSnapTime":curSnapTime})
      snapTimeChangeActive.set(true)
    }
  }

  const getDisplayValues = () => {
    const numTicks = 6;
    const values = [];
    for (let i = 0; i < numTicks; i++) {
      values.push(min + Math.round((max - min) * i / (numTicks - 1)));
    }
    return values;
  }
  $: displayValues = getDisplayValues();
</script>

<div class="wrapper">
  <div class="sel-wrapper">
    {#each displayValues as _, i}
      <div class="sel">
        <div class="top-notch"></div>
        <div class="bot-notch"></div>
      </div>
    {/each}
  </div>
  <div class="num-wrapper">
    {#each displayValues as value}
      <div class="num">
        {value}s
      </div>
    {/each}
  </div>
  <input
    class="slider"
    type="range"
    min={min}
    max={max}
    step="1"
    bind:this={slide}
    bind:value={curSnapTime}>
</div>

<style>
  .wrapper {
    position: relative;
    height: 64px;
  }
  .slider {
    padding: 0 38px 0 38px;
    position: absolute;
    top: 0;
    left: 0;
    -webkit-appearance: none;
    background: none;
    width: 100%;
    height: 64px;
    outline: none;
    -webkit-transition: .2s;
  }
  .slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 48px;
    height: 48px;
    margin-bottom: 4px;
    border: solid 1px #000;
    border-radius: 16px;
    background: #161D17;
    cursor: pointer;
  }
  .slider::-moz-range-thumb {
    width: 48px;
    height: 48px;
    border: solid 1px #000;
    border-radius: 16px;
    background: #161D17;
    cursor: pointer;
  }
  .sel-wrapper {
    position: relative;
    height: 64px;
    display: flex;
    gap: 65px;
    background: #313933;
    border-radius: 16px;
    padding: 0 71px 0 55px;
  }
  .sel {
    flex: 1;
    position: relative;
    height: 64px;
  }
  .num-wrapper {
    display: flex;
    gap: 15px;
    padding: 14px 0 0 38px;
  }
  .num {
    position: relative;
    color: var(--Gray-200, #ABBAAE);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -0.96px;
    width: 50px;
    text-align: center;
  }
  .top-notch {
    width: 16px;
    position: absolute;
    top: 0;
    height: 4px;
    border-radius: 2px;
    background: #5C7060;
  }
  .bot-notch {
    width: 16px;
    position: absolute;
    bottom: 0;
    height: 4px;
    border-radius: 2px;
    background: #5C7060;
  }
</style>