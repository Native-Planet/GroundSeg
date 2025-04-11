<script>
  import { createEventDispatcher } from 'svelte'
  const dispatch = createEventDispatcher()
  export let on = false
  export let loading = false
  let lastUserAction = null;
  let lastActionTime = 0;
  $: effectiveState = shouldIgnoreBackendState() ? lastUserAction : on;
  
  function shouldIgnoreBackendState() {
    return lastUserAction !== null && Date.now() - lastActionTime < 2000;
  }
  
  function handleClick() {
    if (!loading) {
      lastUserAction = !on;
      lastActionTime = Date.now();
      dispatch('click');
    }
  }
  $: if (on === lastUserAction && !loading) {
    lastUserAction = null;
  }
</script>

<div
  class:on={effectiveState} 
  class:loading={loading}
  class="wrapper">
  <div class="text on-text">On</div>
  <div class="text off-text">Off</div>
  <div on:click={handleClick} class="outer">
    <div class="inner" style="margin-left:{effectiveState ? 71 : 8}px"></div>
  </div>
</div>

<style>
  .wrapper {
    position: relative;
    width: 135px;
    height: 65px;
    flex-shrink: 0;
    user-select: none;
    border-radius: 16px;
    background: var(--text-color, #313933);
    cursor: pointer;
    display: inline-flex;
    transition: background-color 0.3s ease;
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
    transition: margin-left 0.3s ease, background-color 0.3s ease;
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
    opacity: 0;
    transition: opacity 0.3s ease;
  }
  .off-text {
    position: absolute;
    right: 12px;
    opacity: 0;
    transition: opacity 0.3s ease;
  }
  .on .on-text {
    opacity: 1;
  }
  .wrapper:not(.on) .off-text {
    opacity: 1;
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
    transition: opacity 0.3s ease;
  }
</style>