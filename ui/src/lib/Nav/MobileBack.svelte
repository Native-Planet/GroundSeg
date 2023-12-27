<script>
  import { goto } from '$app/navigation';
  import { version } from '$lib/stores/display'
  import { createEventDispatcher } from 'svelte'
  export let returnPosition

  const dispatch = createEventDispatcher()
</script>
    <div class="mobile flex">
      <button class="logo" on:click={()=>goto(returnPosition)}>
      </button>
    </div>
<style>
  .mobile {
    height: 90px;
    margin-bottom: 16px;
  }
  .flex {
    display: flex;
    align-items: center;
  }
  .spacer {
    flex: 1;
  }
  .logo {
    margin-left: 32px;
    margin-top: 16px;
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
  .name {
    font-size: 32px;
    font-family: var(--title-font);
    /*color: var(--text-card-color);*/
    leading-trim: both;
    text-edge: cap;
    letter-spacing: -0.72px;
    text-transform: uppercase;
    margin-bottom: 4px;
  }
  .version {
    font-size: 24px;
    line-height: 0px;
  }
  .tabs {
    display: flex;
  }
  .text {
    padding-bottom: 16px;
  }
  .option {
    padding: 20px 32px;
    cursor: pointer;
  }
  .active {
    /*background: var(--bg-base);*/
    background: var(--btn-secondary);
    height: 100%;
    border-radius: 32px;
  }
  .icon {
    height: 48px;
    filter: invert(100);
  }
  .icon-active {
    filter: invert(0);
    /*filter: invert(100);*/
  }
  /*
  .if_ships {
    border-radius: 32px 0 32px 32px;
  }
  */
</style>
