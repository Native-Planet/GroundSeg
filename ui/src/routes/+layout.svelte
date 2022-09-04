<script>
  import Settings from '/src/Components/Buttons/Settings.svelte'
  import { page } from '$app/stores';

</script>

{#if !($page.url.pathname === "/settings")}
  <Settings />
{/if}
<div class="container">
  <div class='slot'>
      <slot />
  </div>
</div>

<style>

  .container {
    position: fixed;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    color: #fff;
    overflow: auto;
    max-height: calc(100vh - 120px);
  }
  
  .slot {
    background: rgba(109, 109, 109, 0.5);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 20px;
    backdrop-filter: blur(60px);
    -moz-backdrop-filter: blur(60px);
    -o-backdrop-filter: blur(60px);
    -webkit-backdrop-filter: blur(60px);
  }
</style>
