<script>
  // Style
  import "../theme.css"
  import { urbitChop, toggleChopAfterVereUpdate } from '$lib/stores/websocket'
  import {  afterUpdate } from 'svelte'
  import { structure, URBIT_MODE } from '$lib/stores/data'
  import { openModal } from 'svelte-modals'
  import ChopModal from '../ChopModal.svelte'

  export let patp

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  $: ship = ($structure?.urbits?.[patp]) || {}
  $: chopOnUpgrade = ship?.info?.chopOnUpgrade == undefined || ship?.info?.chopOnUpgrade // true by default

  $: tChopOnUpgrade = ship?.transition?.chopOnUpgrade || ""
  $: tChop = (ship?.transition.chop) || ""

  const handleModal = () => {
    openModal(ChopModal,{"patp":patp})
  }

  let loading = false
  let awaitChange = false
  let lastState = chopOnUpgrade

  const handleToggleChop = () => {
    loading = true
    awaitChange = true
    lastState = chopOnUpgrade
    toggleChopAfterVereUpdate(patp)
  }

  afterUpdate(()=>{
    if (awaitChange) {
      if (lastState != chopOnUpgrade) {
        loading = false
      }
    }
  })
</script>

<div class="section">
  <div class="section-left">
    <div class="section-title">Chop</div>
    <div class="section-description">This function will trunctate your ship logs, freeing disk space. We recommend configuring automatic chop</div>
    <div class="check-wrapper" class:disabled={loading}>
      <div class="checkbox" on:click={handleToggleChop}>
        {#if chopOnUpgrade}
          <img class="checkmark" src={pfx+"/checkmark-white.svg"} alt="checkmark"/>
        {/if}
      </div>
      <div class="check-text" on:click={handleToggleChop}>Chop after Vere Update</div>
    </div>
  </div>
  <div class="section-right">
    <div class="btn-wrapper">
      <div class="spacer"></div>
      <button
        class:disabled={tChop.length > 0}
        class="super" on:click={()=>urbitChop(patp)}>
        {#if tChop.length < 1 || tChop == "done"}
          Chop
        {:else if tChop == "success"}
          Success!
        {:else if tChop == "error"}
          Error!
        {:else}
          {tChop.charAt(0).toUpperCase() + tChop.slice(1)}
        {/if}
      </button>
      <button class="super chop" on:click={handleModal}>Set Auto</button> 
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
  .super {
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
  .chop {
    background: var(--text-color);  
  }
  .auto {
    padding: 20px 22px;
    background: #313933;
    color: #FFF;
    border-radius: 16px;
  }
  .check-wrapper {
    margin: 32px 0 0 8px;
    display: flex;  
    align-items: center;
    gap: 8px;
  }
  .checkbox {
    width: 24px;
    height: 24px;
    border: solid 1px var(--text-card-color);
    border-radius: 4px;
    cursor: pointer;
  }
  .checkmark {
    width: 16px;
    height: 16px;
    padding: 4px;
    cursor: pointer;
  }
  .check-text {
    font-size: 12px;
    color: var(--text-card-color);
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 150% */
    letter-spacing: -0.96px;
    cursor: pointer;
  }
  .error {color: red;}
  .disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
