<script>
  import { createEventDispatcher, onMount } from 'svelte'
  const dispatch = createEventDispatcher()
  onMount(()=>dispatch("emit"))
  export let uploaded = 0
</script>

<div class="text">Uploading your pier - {uploaded.toFixed(0)}%</div>
<!--
<div class="text">{done} MB / {total} MB</div>
-->

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
