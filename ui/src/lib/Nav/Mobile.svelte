<script>
  import { version } from '$lib/stores/display'
  import { createEventDispatcher } from 'svelte'
  export let tabs
  export let active

  const dispatch = createEventDispatcher()
</script>
    <div class="mobile flex">
      <img alt="groundseg" class="logo" src="/groundseg.svg" />
      <div class="text">
        <div class="name">GROUNDSEG</div>
        <div class="name version">{$version}</div>
      </div>
      <div class="spacer"></div>
      <div class="tabs">
        {#each tabs as tab}
          <button class="option" class:if_ships={tab=="ships"} class:active={tab==active} on:click={()=>dispatch("tabbed",tab)}>
            {#if tab == "startram"}
              <img class="icon startram-icon" class:icon-active={tab==active} src="/satellite-dish-solid.svg" alt="startram icon" />
            {:else if tab == "extensions"}
              <img class="icon extensions-icon" class:icon-active={tab==active} src="/shapes-solid.svg" alt="extensions icon" />
            {:else if tab == "settings"}
              <img class="icon settings-icon" class:icon-active={tab==active} src="/sliders-solid.svg" alt="settings icon" />
            {:else if tab == "ships"}
              <img class="icon ships-icon" class:icon-active={tab==active} src="/igloo-solid.svg" alt="ships icon" />
            {/if}
          </button>
        {/each}
      </div>
    </div>

<style>
  .mobile {
    height: 90px;
    margin-bottom: 16px;
    /*background: var(--btn-secondary);*/
    background: var(--bg-base);
    border-radius: 0 0 32px 32px;
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
    height: 48px;
    margin: 0 16px 0 32px;
    /*filter: invert(100);*/
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
