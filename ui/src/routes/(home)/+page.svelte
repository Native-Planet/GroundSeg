<script>
  import ShipCard from "./ShipCard.svelte"
  import NewShipCard from "./NewShipCard.svelte"
  import { wide } from '$lib/stores/display'

  let ships = [
    "donwer-donlut",
  ]

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
    width: calc((288px * 3) + (80px * 2));
    max-width: 100vw;
  }
  .slim {
    width: calc(100vw - 40px);
  }
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    max-height: calc(90vh - 60px);
    display: flex;
    flex-wrap: wrap;
    padding: 40px;
    gap: 80px;
    overflow: auto;
  }
  .ship {
    flex-basis: 288px;
  }
</style>
