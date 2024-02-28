<script>
  import { goto } from '$app/navigation';
  import { URBIT_MODE } from '$lib/stores/data'
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""
  export let type = ''
</script>

<div class="wrapper" on:click={()=>goto("boot/" + type)} class:disabled={$URBIT_MODE && (type == "existing")}>
  <div class="arrow">
  </div>
  {#if type == "new"}
    <div class="icon"><img alt="import icon" src={pfx+"/newship-icon.svg"} /></div>
    <div class="title">NEW SHIP</div>
    <div class="subtitle">
      Load a new moon, planet, star, or galaxy using your bootfile
    </div>
  {:else if type == "existing"}
    <div class="icon"><img alt="import icon" src={pfx+"/import-icon.svg"} /></div>
    <div class="title">IMPORT PIER</div>
    <div class="subtitle">
      Import an existing pier of a ship
    </div>
  {/if}
</div>

<style>
  .wrapper {
    position: relative;
    background-color: var(--bg-modal);
    color: var(--text-color);
    flex: 1;
    width: 701px;
    border-radius: 24px;
    transition: background-color, 318ms;
    display: flex;
    flex-direction: column;
    padding: 32px;
    cursor: pointer;
  }
  .arrow {
    font-family: var(--title-font);
    background-image: url('/arrow-black.svg');
    background-color: var(--bg-boot-arrow);
    background-repeat: no-repeat;
    background-position: center;
    position: absolute;
    bottom: 0;
    right: 0;
    width: 74px;
    height: 56px;
    border-radius: 24px 0;
  }
  .title {
    font-family: var(--title-font);
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 32px;
    font-style: normal;
    font-weight: 700;
    line-height: 48px; /* 150% */
    letter-spacing: -1.92px;
    text-transform: uppercase;
    margin-bottom: 24px;
  }
  .subtitle {
    width: 416px;
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
  }
  .wrapper:hover {
    background-color: var(--text-card-color);
    cursor: pointer;
  }
  .wrapper:hover .arrow {
    background-image: url('/arrow.svg');
    background-color: var(--text-color);
    cursor: pointer;
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
