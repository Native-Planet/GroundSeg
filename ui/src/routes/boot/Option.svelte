<script>
  import { goto } from '$app/navigation';
  export let type = ''
</script>

<div class="wrapper" on:click={()=>goto("boot/" + type)}>
  <div class="arrow">
  </div>
  {#if type == "new"}
    <div class="icon">+</div>
    <div class="title">NEW SHIP</div>
    <div class="subtitle">
      Load a new moon, planet, star, or galaxy using your bootfile
    </div>
  {:else if type == "existing"}
    <div class="icon">+</div>
    <div class="title">IMPORT PIER</div>
    <div class="subtitle">
      Import an existing pier of a ship
    </div>
  {/if}
</div>

<style>
  .wrapper {
    position: relative;
    background-color: var(--boot-selector-color);
    color: var(--text-color);
    flex: 1;
    width: calc(61.8% - 80px);
    border-radius: 24px;
    transition: background-color, 318ms;
    display: flex;
    flex-direction: column;
    padding: 40px;
    cursor: pointer;
  }
  .arrow {
    font-family: var(--title-font);
    background-image: url('/arrow.svg');
    background-color: var(--bg-card);
    background-repeat: no-repeat;
    background-position: center;
    position: absolute;
    bottom: 0;
    right: 0;
    width: 74px;
    height: 56px;
    border-radius: 24px 0;
  }
  img {
    height: 22px;
    margin: 12px 0;
  
  }
  .icon {
    height: 20px;
    width: 20px;
    /* temp */
    font-size: 20px;
    text-align: center;
    line-height: 20px;
    /* *  * */
  }
  .title {
    font-family: var(--title-font);
    font-size: 24px;
    margin-bottom: 16px;
  }
  .subtitle {
    width: 61.8%;
    font-size: 18px;
    font-weight: 300;
  }
  .wrapper:hover {
    color: var(--text-card-color);
    background-color: var(--bg-card);
    cursor: pointer;
  }
  .wrapper:hover .arrow {
    color: var(--bg-card);
    cursor: pointer;
  }
</style>
