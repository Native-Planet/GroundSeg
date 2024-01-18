<script>
  // Modal
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'
  //import { setPackSchedule, pausePackSchedule } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'

  export let isOpen
  export let patp

  export let maxPierSize = 0
  let newMaxPierSize = maxPierSize == 0 ? 81 : maxPierSize
  $: val = newMaxPierSize === 81 ? 'Unlimited' : newMaxPierSize
  $: hasChanges = maxPierSize === newMaxPierSize
    ? false : (maxPierSize % 81 == 0) && (newMaxPierSize % 81 == 0)
    ? false : true

</script>

<Modal width={640}>
  {#if isOpen}
    <div class="wrapper">
      <div class="header">Automatic Chop</div>
      <div class="item-wrapper">
        <div class="title">Ship size limit</div>
        <div class="description">Automatically chops your ship when it exceeds the size limit</div>
        <div class="display-val">{val}{typeof(val)==="string"?"":" GB"}</div>
        <input type="range" min="20" max="81" bind:value={newMaxPierSize} step="1">
        <button disabled={!hasChanges}>{hasChanges ? "Save Changes" : "No Changes"}</button>
      </div>
      <div class="item-wrapper">
        <div class="title">Roll & Chop</div>
        <div class="description">Rolls your ship into a new epoch before chopping it</div>
        <button>Roll & Chop</button>
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
  input[type="range"] {
    width: 100%;
    height: 64px;
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
</style>
