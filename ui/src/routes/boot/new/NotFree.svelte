<script>
  import { bootShip } from '$lib/stores/websocket';
  import { structure } from '$lib/stores/data'
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { goto } from '$app/navigation';
  import Sigil from './Sigil.svelte'

  import EnvSetup from './EnvSetup.svelte'
  import CreatePier from './CreatePier.svelte'
  import BootingShip from './BootingShip.svelte'
  import SettingRemote from './SettingRemote.svelte'
  import Completed from './Completed.svelte'
  import Aborted from './Aborted.svelte'

  import LogsDrawer from './LogsDrawer.svelte'
  import CancelModal from './CancelModal.svelte'
  import { openModal } from 'svelte-modals'

  export let tBootStage
  let coverage = 0
  $: name = ($structure?.newShip?.transition?.patp) || ""
  $: error = ($structure?.newShip?.transition?.error) || ""
  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)
</script>

<div class="wrapper-not-free">
  <div class="outer" class:error={tBootStage == "aborted"}>
    <div class="back">
      <Sigil {name} swap={true} reverse={true} />
    </div>
    <div class="front">
      <Sigil {name} coverage={coverage} moonbar={false} />
    </div>
  </div>
  <div class="patp">{name.toUpperCase()}</div>
  {#if tBootStage == "starting"}
    <!-- 10% completion -->
    <EnvSetup {name} on:emit={()=>coverage = 0} /> 
  {:else if tBootStage == "creating"}
    <CreatePier {name} on:emit={()=>coverage = 20} /> 
  {:else if tBootStage == "booting"}
    <BootingShip {name} on:emit={()=>coverage = 42} /> 
      <div class="small-button-wrapper">
        <div
          class="small-button cancel"
          on:click={()=>openModal(CancelModal,{"patp":name})}>
          Cancel
        </div>
        <div 
          class="small-button"
          on:click={()=>openModal(LogsDrawer,{"patp":name})}>
          View Logs
        </div>
      </div>
  {:else if tBootStage == "remote"}
    <SettingRemote {name} on:emit={()=>coverage = 86} /> 
      <div class="small-button-wrapper">
        <div
          class="small-button cancel"
          on:click={()=>openModal(CancelModal,{"patp":name})}>
          Cancel
        </div>
        <div 
          class="small-button"
          on:click={()=>openModal(LogsDrawer,{"patp":name})}>
          View Logs
        </div>
      </div>
  {:else if tBootStage == "completed"}
    <Completed {name} on:emit={()=>coverage = 100} /> 
  {:else if tBootStage == "aborted"}
    <Aborted {name} on:emit={()=>coverage = 0} {error} /> 
  {/if}
</div>

<style>
  .wrapper-not-free {
    text-align: center;
  }
  .outer {
    position: relative;
    width: 128px;
    height: 128px;
    border-radius: 16px;
    overflow: hidden;
    margin: auto;
    margin-top: 55px;
  }
  .error {
    border-color: red;
  }
  .back {
    position: absolute;
    left: 0;
  }
  .front {
    position: absolute;
    left: 0;
    top: 0;
  }
  .patp {
    font-size: 42px;
    margin-top: 12px;
    font-family: var(--title-font);
  }
  .small-button-wrapper {
    margin: auto;
    max-width: 240px;
    margin-top: 64px;
    display: flex;
    gap: 48px;
    justify-content: center;
    color: var(--text-color);
  }
  .small-button:hover {
    cursor: pointer;
  }
  .small-button {
    text-decoration: underline;
    font-weight: 600;
  }
  .cancel {
    font-weight: 300;
  }
  .cancel:hover {
    color: red;
  }
</style>
