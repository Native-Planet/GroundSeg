<script>
  import { createEventDispatcher, onMount } from 'svelte'
  const dispatch = createEventDispatcher()
  onMount(()=>dispatch("emit"))
  export let extracted
</script>

<div class="text">Extracting your pier - {extracted}%</div>

<style>
  .text {
    padding-top: 24px;
    font-size: 20px;
    font-weight: 400;
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
