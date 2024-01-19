<script>
  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'
  import { setNewMaxPierSize, urbitRollChop } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'

  export let isOpen
  export let patp


  $: info = ($structure?.urbits?.[patp]?.info)
  $: transition = ($structure?.urbits?.[patp]?.transition)
  $: sizeLimit = info?.sizeLimit || 0
  $: diskUsage = (info?.diskUsage) || 0

  $: tRollChop = transition?.rollChop || ""

  $: newMaxPierSize = sizeLimit == 0 ? 81 : sizeLimit
  $: val = newMaxPierSize === 81 ? 'Unlimited' : newMaxPierSize
  $: hasChanges = sizeLimit === newMaxPierSize
    ? false : (sizeLimit % 81 == 0) && (newMaxPierSize % 81 == 0)
    ? false : true


  const handleNewMaxPierSize = () => {
    setNewMaxPierSize(patp, newMaxPierSize % 81)
  }
</script>

<Modal width={640}>
  {#if isOpen}
    <div class="wrapper">
      <div class="header">Automatic Chop</div>
      <div class="item-wrapper">
        <div class="title">Ship size limit</div>
        <div class="description">Automatically chops your ship when it exceeds the size limit</div>
        <div class="display-val">{(diskUsage / 1024**3).toFixed(2)} GB / {val}{typeof(val)==="string"?"":" GB"}</div>
        <input type="range" min="20" max="81" bind:value={newMaxPierSize} step="1">
        <button disabled={!hasChanges} on:click={handleNewMaxPierSize}>{hasChanges ? "Save Changes" : "No Changes"}</button>
      </div>
      <div class="item-wrapper">
        <div class="title">Roll & Chop</div>
        <div class="description">Rolls your ship into a new epoch before chopping it</div>
        <button 
        class:disabled={tRollChop.length > 0}
        class="super" on:click={()=>urbitRollChop(patp)}>
        {#if tRollChop.length < 1 || tRollChop == "done"}
          Roll & Chop
        {:else if tRollChop == "success"}
          Success!
        {:else if tRollChop == "error"}
          Error!
        {:else}
          {tRollChop.charAt(0).toUpperCase() + tRollChop.slice(1)}
        {/if}
      </div>
    </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
    display: flex;
    flex-direction: column;
    gap: 32px;
  }
  .header {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .title {
    display: flex;
    gap: 16px;
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    max-width: 460px;
    align-items: center;
    margin: 0;
  }
  .description {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    line-height: 55px;
    min-width: 55px;
  }
  .display-val {
    width: 100%;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;

  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: black;
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    cursor: pointer;
    height: 65px;
  }
  button:disabled {
    opacity: .6;
    pointer-events:none;
  }
  .disabled {
    pointer-events: none;
    opacity: .6;
  }
  input[type="range"] {
    -webkit-appearance: none; /* Override default CSS styles */
    appearance: none;
    width: 100%;
    margin: 16px 0;
  }
  /* Thumb */
  input[type="range"]::-webkit-slider-thumb {
    -webkit-appearance: none; /* Override default look */
    appearance: none;
    width: 62px; /* Width of the thumb */
    height:62px; /* Height of the thumb */
    background: var(--btn-secondary);
    border-radius: 16px;
    border: solid 1px black;
    cursor: pointer; /* Cursor on hover */
  }

  input[type="range"]::-moz-range-thumb {
    -webkit-appearance: none; /* Override default look */
    appearance: none;
    width: 62px; /* Width of the thumb */
    height:62px; /* Height of the thumb */
    background: var(--btn-secondary);
    border-radius: 16px;
    border: solid 1px black;
    cursor: pointer; /* Cursor on hover */
  }

  input[type="range"]::-ms-thumb {
    -webkit-appearance: none; /* Override default look */
    appearance: none;
    width: 62px; /* Width of the thumb */
    height:62px; /* Height of the thumb */
    background: var(--btn-secondary);
    border-radius: 16px;
    border: solid 1px black;
    cursor: pointer; /* Cursor on hover */
  }

  /* Track */
  input[type="range"]::-webkit-slider-runnable-track {
    width: 100%; /* Width of the track */
    height: 64px; /* Height of the track */
    background: var(--bg-base); /* Track background */
    border-radius: 16px; /* Roundness of the track */
  }

  input[type="range"]::-moz-range-track {
    width: 100%; /* Width of the track */
    height: 64px; /* Height of the track */
    background: var(--bg-base); /* Track background */
    border-radius: 16px; /* Roundness of the track */
  }

  input[type="range"]::-ms-track {
    width: 100%; /* Width of the track */
    height: 64px; /* Height of the track */
    background: var(--bg-base); /* Track background */
    border-radius: 16px; /* Roundness of the track */
  }
</style>
