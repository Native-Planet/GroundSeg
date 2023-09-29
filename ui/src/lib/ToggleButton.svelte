<script>
  import { createEventDispatcher } from 'svelte'
  const dispatch = createEventDispatcher()
  export let on = false
  export let loading = false

  /* debug
  let on = false
  const loop = () => {on = !on;setTimeout(loop,3000)}
  loop()
  */
</script>

<div
  class:on={on} 
  class:loading={loading}
  class="wrapper">
  <div class="text on-text">On</div>
  <div class="text off-text">Off</div>
  <div on:click={()=>dispatch('click')} class="outer">
    <div class="inner" style="margin-left:{on ? 71 : 8}px"></div>
  </div>
</div>

<style>
  .wrapper {
    position: relative;
    width: 135px;
    height: 65px;
    flex-shrink: 0;

    border-radius: 16px;
    background: var(--text-color, #313933);
    cursor: pointer;
    display: inline-flex;
  }
  .outer {
    position: absolute;
    align-items: center;
    width: 100%;
    height: 100%;
  }
  .inner {
    width: 56px;
    height: 49px;
    border-radius: 10px;
    background: #161D17;
    transition: margin-left 0.2s ease;
    margin-top: 8px;
  }
  .on {
    background: #077D13;
  }
  .on > .outer > .inner {
    background: #07D91C;
  }
  .on-text {
    position: absolute;
    left: 12px;
  }
  .off-text {
    position: absolute;
    right: 12px;
  }
  .text {
    top: 16px;
    color: var(--NP_White, #F8F8F6);
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    width: 47px;
    height: 47px;
  }
  .loading {
    opacity: .6;
    pointer-events: none;
  }
</style>
