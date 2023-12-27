<script>
  import { version } from '$lib/stores/display'
  import { createEventDispatcher } from 'svelte'
  export let tabs
  export let active

  const dispatch = createEventDispatcher()
</script>
    <div class="desktop flex">
      <img alt="groundseg" class="logo" src="/groundseg.svg" />
      <div class="text">
        <div class="name">GROUNDSEG</div>
        <div class="name version">{$version}</div>
      </div>
      <div class="spacer"></div>
      <div class="tabs">
        {#each tabs as tab}
          <button class="name option" class:if_ships={tab=="ships"} class:active={tab==active} on:click={()=>dispatch("tabbed",tab)}>{tab}</button>
        {/each}
      </div>
    </div>

<style>
  .desktop {
    height: 64px;
    margin: 16px auto;
    max-width: 1200px;
    background: var(--bg-base);
    border-radius: 16px;
    overflow: hidden;
  }
  .flex {
    display: flex;
    align-items: center;
  }
  .spacer {
    flex: 1;
  }
  .logo {
    height: 32px;
    margin: 0 16px 0 32px;
  }
  .text {
    padding: 6px 0 28px 0;
  }
  .name {
    font-size: 32px;
    font-family: var(--title-font);
    leading-trim: both;
    text-edge: cap;
    letter-spacing: -0.72px;
    text-transform: uppercase;
  }
  .version {
    font-size: 16px;
    line-height: 0px;
  }
  .tabs {
    display: flex;
  }
  .option {
    padding: 0 16px 8px 16px; 
    cursor: pointer;
  }
  .active {
    line-height: 56px;
    background: var(--btn-secondary);
    height: 100%;
    border-radius: 32px 0 32px 0;
    color: white;
  }
  .if_ships {
    border-radius: 32px 0 16px 0;
  }
</style>
