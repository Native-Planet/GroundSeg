<script>
  import { goto } from '$app/navigation';
  import { page } from '$app/stores'
  import { wide, version } from '$lib/stores/display'
</script>

<div class="wrapper {wide ? "wide" : "slim"}">
  <div class="version">{$version}</div>
  <div class="app">GROUNDSEG</div>
  <div class="nav">
    <div class="ships">
      <div
        class:highlight={$page.route.id != "/(home)"}
        on:click={()=>goto("/")}
        class="btn text"
        >SHIPS
      </div>
      <div
        on:click={()=>goto("/boot")}
        class="btn add"
        >
       +
      </div>
    </div>
    <div
      class:highlight={$page.route.id != "/profile"}
      on:click={()=>goto("/profile")}
      class="btn option"
      >PROFILE
    </div>
    <div
      class:highlight={$page.route.id != "/system"}
      on:click={()=>goto("/system")}
      class="btn option"
      >SYSTEM
    </div>
  </div>
</div>

<style>
  .wide {
    width: calc((288px * 3) + (120px * 2));
    max-width: 98vw;
  }
  .slim {
    width: 100vw;
  }
  .wrapper {
    color: var(--text-color);
    margin: auto;
    margin-top: 10px;
  }
  .version {
    font-family: var(--title-font);
    font-size: 14px;
  }
  .app {
    font-family: var(--title-font);
    font-size: 16px;
    margin-bottom: 16px;
  }
  .nav {
    display: flex;
    margin-bottom: 10px;
    gap: 24px;
  }
  .ships {
    flex: 1;
    display: flex;
    align-items: end;
    gap: 10px;
  }
  .text {
    font-family: var(--title-font);
    font-size: 28px;
  }
  .add {
    background: #06A315;
    width: 20px;
    height: 20px;
    margin-bottom:4px;
    border-radius: 100%;
  }
  .option {
    font-family: var(--title-font);
    font-size: 28px;
  }
  .btn:hover {
    cursor: pointer;
  }
  .highlight {
    opacity: .8;
  }
</style>
