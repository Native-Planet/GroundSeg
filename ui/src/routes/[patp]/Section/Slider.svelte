<script>
  export let min = 28
  export let max = 33
  let x = 0

  $: marginLeft = fixMargin(x)

  const fixMargin = pos => {
    let m = 38
    if (pos < m) {
      return m
    } else if ((pos > m) && (pos < (m+65))) {
      // if closer to right
      if ((m + 65 - pos - 25) >= (m - pos))  {
        return pos - 25
      }
    } 
    // check if pos is closer to 
    pos = pos - 25
    return pos
  }
    // by default there will always be 38px on the left

  const mouseHandler = e => {
    const rect = e.currentTarget.getBoundingClientRect()
    x = e.clientX - rect.x
  }
</script>

<div on:mousemove={mouseHandler}>
  <div class="sel-wrapper">
    {#each Array.from({ length: (max-min+1) }, (_, i) => i) as i}
      <div class="sel">
        <div class="top-notch"></div>
        <div class="bot-notch"></div>
      </div>
    {/each}
    <div class="thumb" style="margin-left: {marginLeft}px;"></div>
  </div>
  <div class="num-wrapper">
    {#each Array.from({ length: (max-min+1) }, (_, i) => i) as i}
      <div class="num">
        {2**(i+min)/(1024*1024)}
      </div>
    {/each}
  </div>
</div>

<style>
  .sel-wrapper {
    position: relative;
    height: 64px;
    display: flex;
    gap: 65px;
    background: #313933;
    border-radius: 16px;
    padding: 0 71px 0 55px;
  }
  .thumb {
    position: absolute;
    width: 48px;
    height: 48px;
    background: #161D17;
    border-radius: 16px;
    border: 1px solid black;
    top: 8px;
    left: 0;
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
