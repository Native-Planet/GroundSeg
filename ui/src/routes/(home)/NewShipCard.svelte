<script>
  import { goto } from '$app/navigation'
  import { URBIT_MODE } from '$lib/stores/data'
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""
</script>


<div class="wrapper">
  <div class="ship" on:click={()=>goto(pfx+"/boot")}>
    <div class="sigil"></div>
    <div class="bg"></div>
  </div>
  <div class="add" on:click={()=>goto(pfx+"/boot")}>+</div>
</div>

<style>
  .wrapper {
    position: relative;
    width: 320px;
    height: 180px;
    flex-shrink: 0;
  }
  .add {
    position: absolute;
    font-family: var(--title-font);
    color: var(--text-card-color);
    font-size: 64px;
    width: 40px;
    top: 40px;
    left: calc(50% - 15px);
    cursor: pointer;
  }
  .ship {
    position: absolute;
    width: 100%;
    height: 100%;
    opacity: .6;
  }
  .wrapper:hover > .ship {
    opacity: 1;
    cursor: pointer;
  }
  .bg {
    position: absolute;
    width: 100%;
    height: 100%;
    background-image: url('/ship.svg');
    background-position: center;
    background-repeat: no-repeat;
    background-size: cover;
  }
  .sigil {
    position: absolute;
    background: var(--btn-secondary);
    width: 72px;
    height: 58px;
    flex-shrink: 0;
    left: 9px;
    border-radius: 8px 0;
    top: 6px;
  }
</style>
