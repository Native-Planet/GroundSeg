<script>
  import Sigil from '$lib/Sigil.svelte'
  import KeyDropper from './KeyDropper.svelte';
  import { bootShip, structure } from '$lib/stores/websocket';
  import { sigRemove, checkPatp } from '$lib/stores/patp';
  import { goto } from '$app/navigation';
  import EnvSetup from './EnvSetup.svelte'

  export let tBootStage
  let coverage = 0

  $: name = ($structure?.newShip?.transition?.patp) || ""
  $: noSig = sigRemove(name)
  $: validPatp = checkPatp(noSig)

  import { onMount } from 'svelte'
  onMount(()=>timerCount())
  const timerCount = () => {
    coverage = coverage + 10
    setTimeout(timerCount,1000)
  }
</script>

<div class="outer">
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
{/if}

<style>
  .outer {
    position: relative;
    width: 160px;
    height: 160px;
    border: solid 4px var(--text-color);
    border-radius: 24px;
    overflow: hidden;
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
