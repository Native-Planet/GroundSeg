<script>
  import { goto } from '$app/navigation';
  import { page } from '$app/stores'
  import { wide, version } from '$lib/stores/display'
  import { structure, URBIT_MODE } from '$lib/stores/data'
  import SatelliteDish from '$lib/SatelliteDish.svelte'
  import DiskWarningIcon from '$lib/DiskWarningIcon.svelte'

  // Temp dev mode
  import DevToggle from '$lib/DevToggle.svelte'

  $: registered = ($structure?.profile?.startram?.info?.registered) || false
  $: running = ($structure?.profile?.startram?.info?.running) || false
  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""
  $: authLevel = ($structure?.auth_level) || "unauthorized"

  const handleBack = () => {
    const bootExist = $page.route.id.includes("new")
    const bootNew = $page.route.id.includes("existing")
    if (bootExist || bootNew) {
      goto(pfx+"/boot")
    } else {
      goto(pfx+"/")
    }
  }

</script>

<div class="wrapper {wide ? "wide" : "slim"}">
  {#if ($page.route.id == '/[patp]') || ($page.route.id.includes('/boot'))}
    {#if authLevel == "authorized"}
      <div class="back" on:click={handleBack}></div>
    {/if}
  {:else}
    <div class="nav">
      <div class="status-bar">
        <div class="app">
          <div class="app-name">GROUNDSEG</div>
          <div class="app-version">{$version}</div>
        </div>
        <SatelliteDish />
        <DiskWarningIcon />
          <!--
        <DevToggle />
          -->
      </div>
      <div class="spacer"></div>
      <div
        class:highlight={$page.route.id != (pfx+"/(home)")}
        on:click={()=>goto(pfx+"/")}
        class="btn option"
        >SHIPS
      </div>
      <div
        class:highlight={$page.route.id != (pfx+"/apps")}
        on:click={()=>goto(pfx+"/apps")}
        class="btn option"
        >APPS
      </div>
      <div
        class:highlight={$page.route.id != (pfx+"/profile")}
        on:click={()=>goto(pfx+"/profile")}
        class="btn option"
        >PROFILE
      </div>
      <div
        class:highlight={$page.route.id != (pfx+"/system")}
        on:click={()=>goto(pfx+"/system")}
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
  }
  .nav {
    display: flex;
    margin-top: 4px;
    gap: 24px;
    align-items: end;
    margin-bottom: 16px;
  }
  .spacer {
    flex: 1;
  }
  .status-bar {
    display: flex;
    gap: 16px;
    align-items: end;
  }
  .app {
    position: relative;
    height: 32px;
    width: 180px;
  }
  .app-name {
    position: absolute;
    font-family: var(--title-font);
    font-size: 32px;
    top: -15px;
    left: 0;
  }
  .app-version {
    position: absolute;
    font-family: var(--title-font);
    font-size: 14px;
    bottom: 0;
    left: 0;
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
    margin-top: 8px;
  }
  .back:hover {
    cursor: pointer;
    background-color: var(--bg-card);
  }
  .ships {
    flex: 1;
    display: flex;
    align-items: center;
    margin-left: 8px;
  }
  .option {
    font-size: 32px;
    font-family: var(--title-font);
    opacity: .2;
    pointer-events: none;
  }
  .highlight {
    opacity: 1;
    cursor: pointer;
    pointer-events: auto;
  }
</style>
