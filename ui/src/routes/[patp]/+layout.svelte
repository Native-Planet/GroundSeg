<script>
  import Nav from '$lib/nav.svelte'
  import { Modals, closeModal } from 'svelte-modals'
</script>

<Nav />
<slot />

<Modals>
  <div
    slot="backdrop"
    class="backdrop"
    on:click={closeModal}
  />
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
