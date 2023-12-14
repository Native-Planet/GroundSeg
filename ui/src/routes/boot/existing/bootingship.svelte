<script>
  import { createEventDispatcher, onMount } from 'svelte'
  const dispatch = createEventDispatcher()
  onMount(()=>dispatch("emit"))
  export let name = ""
</script>

<div class="text">Booting...</div>
<div class="subtext">This may take a while</div>

<style>
  .text {
    padding: 24px;
    font-size: 20px;
    font-weight: 400;
    color: var(--text-color);
    animation: breathe 5s infinite;
  }
  .subtext {
    font-size: 12px;
    font-weight: 300;
    color: var(--text-color);
    animation: breathe 5s infinite;
  }
  @keyframes breathe {
    0% {
      opacity: .2;
    }
    50% {
      opacity: 1;
    }
    100% {
      opacity: .2;
    }
  }
</style>
