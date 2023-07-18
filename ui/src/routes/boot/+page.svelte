<script>
  import { wide } from '$lib/stores/display';
  import { goto } from '$app/navigation';

  import Option from './Option.svelte';

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="title">LOAD A SHIP</div>
  <Option type="new" />
  <Option type="existing"/>
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
    height: 70vh;
    padding: 40px;
    display:flex;
    gap: 40px;
    flex-direction: column;
    align-items: center;
  }
  .title {
    font-family: var(--title-font);
    font-size: 48px;
  }
  .selection {
  }
</style>
