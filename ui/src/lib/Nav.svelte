<script>
  import { goto } from '$app/navigation';
  import { page } from '$app/stores'
  import { wide, version } from '$lib/stores/display'
  import { structure } from '$lib/stores/websocket'

  $: registered = ($structure?.startram?.registered) || false

</script>

<div class="wrapper {wide ? "wide" : "slim"}">
  <div class="app">GROUNDSEG {$version}{registered ? " - STARTRAM" : ""}</div>
  {#if ($page.route.id == '/[patp]') || ($page.route.id.includes('/boot'))}
    <div class="back" on:click={()=>goto("/")}>
    </div>
  {:else}
    <div class="nav">
      <div class="ships">
        <div
          class:highlight={$page.route.id != "/(home)"}
          on:click={()=>goto("/")}
          class="btn text"
          >SHIPS
        </div>
      </div>
      <div
        class:highlight={$page.route.id != "/support"}
        on:click={()=>goto("/support")}
        class="btn option"
        >SUPPORT
      </div>
      <div
        class:highlight={$page.route.id != "/logs"}
        on:click={()=>goto("/logs")}
        class="btn option"
        >LOGS
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
  {/if}
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
  .back {
    width: 48px;
    height: 48px;
    margin-bottom: 12px;
    background-image: url('/arrow.svg');
    background-repeat: no-repeat;
    background-position: center;
    background-color: var(--btn-secondary);
    border-radius: 16px 0;
    transform: rotate(180deg);
  }
  .back:hover {
    cursor: pointer;
    background-color: var(--bg-card);
  }
  .ships {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .text {
    font-family: var(--title-font);
    font-size: 28px;
  }
  .option {
    font-size: 28px;
    font-family: var(--title-font);
  }
  .btn:hover {
    cursor: pointer;
  }
  .highlight {
    opacity: .6;
  }
</style>
