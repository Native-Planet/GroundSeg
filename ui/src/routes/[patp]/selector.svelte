<script>
  import Fa from 'svelte-fa'
  import { faAngleUp, faAngleDown } from '@fortawesome/free-solid-svg-icons';
  import { createEventDispatcher } from 'svelte'

  export let num
  export let packIntervalType

  let currentOption = packIntervalType.charAt(0).toUpperCase() + packIntervalType.slice(1)
  let optionMenu = false

  const dispatch = createEventDispatcher()

  let selectOptions = ["Day","Week","Month"]
  const handleSelect = selected => {
    currentOption = selected  
    optionMenu = false
    dispatch("change",currentOption)
  }
</script>
<div class="wrapper">
  <div class="selector" on:click={()=>optionMenu=!optionMenu}>
    {currentOption}{#if num > 1}s{/if}
      {#if optionMenu}
        <span class="icon"><Fa icon={faAngleUp} size="1x" /></span>
      {:else}
        <span class="icon"><Fa icon={faAngleDown} size="1x" /></span>
      {/if}
  </div>
  {#if optionMenu}
  <div class="options">
    {#each selectOptions as sel}
      <div class="option" on:click={()=>handleSelect(sel)}>{sel}{#if num > 1}s{/if}</div>
    {/each}
  </div>
  {/if}
</div>

<style>
  .wrapper {
    height: 55px;
    width: 240px;
    position: relative;
    display: flex;
    align-items: center;
    z-index: 1;
  }
  .selector {
    user-select: none;
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
    width: 100%;
    padding: 0 24px;
    cursor: pointer;
  }
  .options {
    border-radius: 16px;
    display: flex;
    flex-direction: column;
    gap: 16px;
    position: absolute;
    top: 64px;
    width: calc(100% - 48px);
    background: var(--bg-modal);
    padding: 12px 24px;
    border: solid 1px var(--btn-secondary);
  }
  .option {
    cursor: pointer;
  }
  .option:hover {
    font-weight: 500;
  }
  span {
    float: right;
  }
</style>
