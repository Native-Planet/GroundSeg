<script>
  import { goto } from '$app/navigation';
  import { version } from '$lib/stores/display'
  import { createEventDispatcher } from 'svelte'
  export let returnPosition = "/"

  const dispatch = createEventDispatcher()
</script>
    <div class="desktop flex">
      <button class="logo" on:click={()=>goto(returnPosition)}>
      </button>
    </div>

<style>
  .desktop {
    height: 64px;
    margin: 16px auto;
    max-width: 1200px;
    overflow: hidden;
  }
  .flex {
    display: flex;
    align-items: center;
  }
  .logo {
    width: 64px;
    height: 64px;
    background-image: url('/arrow.svg');
    background-repeat: no-repeat;
    background-position: center;
    background-color: var(--btn-secondary);
    border-radius: 16px 0;
    transform: rotate(180deg);
  }
  .logo:hover {
    cursor: pointer;
    background-color: var(--bg-card);
  }
</style>
