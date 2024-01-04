<script>
  import FinalModal from './Section/FinalModal.svelte';
  import UnplugWarning from './Section/UnplugWarning.svelte';
  import { afterUpdate } from 'svelte'
  import { rebuildContainer } from '$lib/stores/websocket'
  import { URBIT_MODE, structure } from '$lib/stores/data'
  import LogsDrawer from './LogsDrawer.svelte'
  import DeleteModal from './DeleteModal.svelte'
  import ExportModal from './ExportModal.svelte'
  import { openModal } from 'svelte-modals'
  export let patp

  $: tRebuildContainer = ($structure?.urbits?.[patp]?.transition?.rebuildContainer) || ""
  $: t = tRebuildContainer

  let ownShip = false

  afterUpdate(()=>{
    if ($URBIT_MODE) {
      ownShip = (window.ship == patp)
    }
  })

  function handleClick() {
    if ($URBIT_MODE) {
      openModal(FinalModal, {"component":"rebuild","patp":patp})
    } else {
      rebuildContainer(patp)
    }
  }

</script>
<div class="bottom-panel">
  {#if !$URBIT_MODE}
    <button 
      class="btn" 
      on:click={()=>openModal(LogsDrawer,{"patp":patp})}>
      Logs
    </button>
  {/if}
  <div class="spacer"></div>
  <UnplugWarning component="rebuild" {ownShip} >
    <div class="btn rebuild" class:disabled={t.length > 0} on:click={handleClick}>
      {#if t.length < 1}
        Rebuild
      {:else if t == "loading"}
        Rebuilding
      {:else if t == "success"}
        Success!
      {:else if t == "error"}
        Error
      {/if}
    </div>
  </UnplugWarning>
  {#if !ownShip}
    <button 
      class="btn" 
      on:click={()=>openModal(ExportModal,{"patp":patp})}>
      Export
    </button>
    <button 
      class="btn" 
      on:click={()=>openModal(DeleteModal,{"patp":patp})}>
      Delete
    </button>
  {/if}
</div>
<style>
  .bottom-panel {
    display: flex;
    bottom: 0;
    gap: 12px;
    width: 85%;
  }
  .btn {
    color: var(--NP_White, #F8F8F6);
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    letter-spacing: -1.44px;

    color: var(--text-card-color);
    background-color: var(--text-color);
    cursor: pointer;
    border-radius: 16px 16px 0px 0px;

    display: inline-flex;
    padding: 16px 32px;
    justify-content: center;
    align-items: center;
    gap: 8px;
  }
  .rebuild {
    background-color: var(--fg-card);
  }
  .disabled {
    opacity: .6;
    pointer-events: none;
  }
  .spacer {
    flex: 1;
  }
</style>
