<script>
  /*
  import KeyDropper from './KeyDropper.svelte';
  import { bootShip } from '$lib/stores/websocket';
  import { structure } from '$lib/stores/data'
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { goto } from '$app/navigation';

  $: name = ($structure?.newShip?.transition?.patp) || ""
  $: error = ($structure?.newShip?.transition?.error) || ""
  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)
*/
  import Sigil from './sigil.svelte'

  import Uploading from './uploading.svelte'
  import CreatePier from './createpier.svelte'
  import Extracting from './extracting.svelte'
  import BootingShip from './bootingship.svelte'
  import SettingRemote from './settingremote.svelte'
  import Completed from './completed.svelte'
  import Aborted from './aborted.svelte'

  export let status
  export let name
  export let error
  export let uploaded
  export let extracted

  let coverage = 0
</script>

<div class="wrapper-not-free">
  <div class="outer" class:error={status == "aborted"}>
    <div class="back">
      <Sigil {name} swap={true} reverse={true} />
    </div>
    <div class="front">
      <Sigil {name} coverage={coverage} moonbar={false} />
    </div>
  </div>
  <div class="patp">{name.toUpperCase()}</div>
  {#if status == "uploading"}
    <Uploading {uploaded} on:emit={()=>coverage = 20} /> 
  {:else if status == "creating"}
    <CreatePier {name} on:emit={()=>coverage = 60} /> 
  {:else if status == "extracting"}
    <Extracting {name} {extracted} on:emit={()=>coverage = 78} /> 
  {:else if status == "booting"}
    <BootingShip {name} on:emit={()=>coverage = 86} /> 
  {:else if status == "remote"}
    <SettingRemote {name} on:emit={()=>coverage = 92} /> 
  {:else if status == "completed"}
    <Completed {name} on:emit={()=>coverage = 100} /> 
  {:else if status == "aborted"}
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
</style>
