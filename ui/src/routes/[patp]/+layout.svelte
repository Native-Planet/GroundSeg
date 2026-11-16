<script>
  import Nav from '#lib/Nav.svelte'
  import { Modals, modals } from 'svelte-modals'
</script>

<Nav />
<slot />

<Modals>
  {#snippet backdrop()}
    <div class="backdrop" on:click={() => modals.close()} />
  {/snippet}
</Modals>

<style>
  .backdrop {
    position: fixed;
    top: 0;
    bottom: 0;
    right: 0;
    left: 0;
    background: rgba(92, 112, 96, 0.5);
  }
</style>
