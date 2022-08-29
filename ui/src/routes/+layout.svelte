<script>
  import Settings from '/src/Components/Buttons/Settings.svelte'
</script>

<Settings />
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
