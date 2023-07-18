<script>
  import ShipCard from "$lib/ShipCard.svelte"
  import { wide } from '$lib/stores/display'

  let ships = [
    "donwer-donlut",
    "famnyx-walsur",
    "padnym-fodsub",
    "morpub-fotdul",
    "pasnep-norful",
    "lomwes-nodfer",
    "ramtec-ladnus"
  ]

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
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
