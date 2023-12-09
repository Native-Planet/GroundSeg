<script>
  import ShipCard from "./ShipCard.svelte"
  import NewShipCard from "./NewShipCard.svelte"
  import { wide } from '$lib/stores/display'
  import { structure } from '$lib/stores/websocket'
  import { sortModes } from '$lib/stores/patp'

  const sortMode = 'hierarchical'
  
  $: urbits = ($structure?.urbits) || {}
  $: ships = sortModes[sortMode](Object.keys(urbits))

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
  <NewShipCard />
  {#each ships as p}
    <div class="ship">
      <ShipCard patp={p} />
    </div>
  {/each}
</div>

<style>
  .wide {
    width: calc((320px * 3) + (80px * 2));
    max-width: 100vw;
  }
  .slim {
    width: calc(100vw - 40px);
  }
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    display: flex;
    min-height: calc(520px - 80px);
    flex-wrap: wrap;
    padding: 40px;
    gap: 80px;
  }
  .ship {
    flex-basis: 288px;
    height: 180px;
  }
</style>