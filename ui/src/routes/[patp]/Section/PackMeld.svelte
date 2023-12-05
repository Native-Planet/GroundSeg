<script>
  // Style
  import "../theme.css"
  import { openModal } from 'svelte-modals'
  import PackScheduleModal from '../PackScheduleModal.svelte'
  import { urthPackMeld, marsPack } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import { createEventDispatcher } from 'svelte'
  import { URBIT_MODE } from '$lib/stores/data'
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  export let patp

  const dispatch = createEventDispatcher()


  const handleModal = () => {
    openModal(PackScheduleModal,{"patp":patp})
  }

  $: tPack = ($structure?.urbits?.[patp]?.transition?.pack) || ""
  $: tPackMeld = ($structure?.urbits?.[patp]?.transition?.packMeld) || ""

</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Pack Pier</div>
    <div class="section-description">
      This function will refragement your ship's memory capacity, optimizing its performance. We recommend scheduling these once a week
    </div>
  </div>
  <div class="section-right">
    <div class="btn-wrapper">
      <div class="spacer"></div>
      <button disabled={tPackMeld.length > 0} class="start urth" on:click={()=>urthPackMeld(patp)}>
        {#if tPackMeld.length < 1}
          Pack & Meld
        {:else if tPackMeld == "stopping"}
          Getting ready
        {:else if tPackMeld == "packing"}
          Packing..
        {:else if tPackMeld == "melding"}
          Melding..
        {:else if tPackMeld == "starting"}
          Starting ship
        {:else if tPackMeld == "success"}
          Success!
        {:else}
          Failed :(
        {/if}
      </button>
      <button disabled={tPack.length > 0} class="start" on:click={()=>marsPack(patp)}>
        {#if tPack.length < 1}
          Pack
        {:else if tPack == "packing"}
          Packing..
        {:else if tPack == "success"}
          Success!
        {:else}
          Failed :(
        {/if}
      </button>
      <button class="calendar" on:click={handleModal}>
        <img src={pfx+"/calendar.svg"} alt="calendar icon" width="20px" height="20px"/>
      </button>
    </div>
  </div>
</div>

<style>
  .btn-wrapper {
    display: flex; 
    gap: 8px;
  }
  .spacer {
    flex: 1;
  }
  button {
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .start {
    display: flex;
    padding: 20px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    border-radius: 16px;
    background: #2C3A2E;
    color: #FFF;
    font-family: Inter;
    font-size: 32px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 75% */
    letter-spacing: -1.92px;
  }
  .urth {
    background: var(--text-color);  
  }
  .calendar {
    padding: 20px 22px;
    background: #313933;
    color: #FFF;
    border-radius: 16px;
  }
</style>
