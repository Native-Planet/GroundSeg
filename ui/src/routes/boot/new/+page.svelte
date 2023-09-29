<script>
  import { wide } from '$lib/stores/display';
  import { structure } from '$lib/stores/websocket';
  import Free from './Free.svelte'
  import NotFree from './NotFree.svelte'

  $: tBootStage = ($structure?.newShip?.transition?.bootStage) || ""
</script>

<div id="card-wrapper" class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title-wrapper">
    <div class="title">NEW SHIP</div>
  </div>
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
    margin: auto;
    width: calc(1173px - 160px);
    border-radius: 16px;
    flex-shrink: 0;
    padding: 80px;
  }
  .title-wrapper {
    overflow: hidden;
  }
  .title {
    position: relative;
    top: -19px;
    color: #000;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 48px;
    line-height: 47px;
    font-style: normal;
    font-weight: 700;
    letter-spacing: -2.88px;
    text-transform: uppercase;
  }
</style>
