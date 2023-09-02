<script>
  import { wide } from '$lib/stores/display';
  import { structure } from '$lib/stores/websocket';
  import Free from './Free.svelte'
  import NotFree from './NotFree.svelte'

  $: tBootStage = ($structure?.newShip?.transition?.bootStage) || ""

</script>

<div id="card-wrapper" class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title">NEW SHIP</div>
  {#if tBootStage.length < 1}
    <Free />
  {:else}
    <NotFree {tBootStage} />
  {/if}
</div>

<style>
  .wide {
    width: 1104px;
    max-width: 100vw;
  }
  .slim {
    width: calc(100vw - 40px);
  }
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    display:flex;
    flex-direction: column;
    align-items: center;
  }
  .slim .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 70vh;
    display:flex;
    gap: 40px;
    flex-direction: column;
    align-items: center;
  }
  .title {
    font-family: var(--title-font);
    font-size: 48px;
    padding-top: 40px;
    padding-bottom: 40px;
  }
</style>
