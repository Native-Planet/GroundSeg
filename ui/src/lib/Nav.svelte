<script>
  import { goto } from '$app/navigation';
  import { page } from '$app/state'
  import { wide, version } from '#lib/stores/display.js'
  import { structure, URBIT_MODE } from '#lib/stores/data.js'
  import SatelliteDish from '#lib/SatelliteDish.svelte'
  import DiskWarningIcon from '#lib/DiskWarningIcon.svelte'

  // Temp dev mode
  import DevToggle from '#lib/DevToggle.svelte'

  let pfx = $derived($URBIT_MODE ? "/apps/groundseg" : "")
  let authLevel = $derived(($structure?.auth_level) || "unauthorized")
  let hostShip = $derived($URBIT_MODE && typeof window !== 'undefined' ? window.ship : '')
  let shipTabPath = $derived($URBIT_MODE && hostShip ? `${pfx}/${hostShip}` : `${pfx}/`)
  let shipTabLabel = $derived($URBIT_MODE ? "SHIP" : "SHIPS")
  let routeId = $derived(page.route.id || "")
  let shipTabCurrent = $derived($URBIT_MODE
    ? routeId == "/[patp]"
    : routeId == (pfx+"/(home)"))
  let keysTabCurrent = $derived((routeId == "/apps") || (routeId == (pfx+"/apps")))
  let profileTabCurrent = $derived((routeId == "/profile") || (routeId == (pfx+"/profile")))
  let systemTabCurrent = $derived((routeId == "/system") || (routeId == (pfx+"/system")))

  const handleBack = () => {
    const bootExist = page.route.id.includes("new")
    const bootNew = page.route.id.includes("existing")
    if (bootExist || bootNew) {
      goto(pfx+"/boot")
    } else {
      goto(pfx+"/")
    }
  }

</script>

<div class="wrapper {wide ? "wide" : "slim"}">
  {#if (!$URBIT_MODE && (page.route.id == '/[patp]')) || (page.route.id.includes('/boot'))}
    {#if authLevel == "authorized"}
      <div class="back" onclick={handleBack}></div>
    {/if}
  {:else}
    <div class="nav">
      <div class="status-bar">
        <div class="app">
          <div class="app-name">GROUNDSEG</div>
          <div class="app-version">{$version}</div>
        </div>
        {#if !$URBIT_MODE}
          <SatelliteDish />
          <DiskWarningIcon />
        {/if}
          <!--
        <DevToggle />
          -->
      </div>
      <div class="spacer"></div>
      <div
        class:highlight={!shipTabCurrent}
        onclick={()=>goto(shipTabPath)}
        class="btn option"
        >{shipTabLabel}
      </div>
      <div
        class:highlight={!keysTabCurrent}
        onclick={()=>goto(pfx+"/apps")}
        class="btn option"
        >KEYS
      </div>
      {#if !$URBIT_MODE}
        <div
          class:highlight={!profileTabCurrent}
          onclick={()=>goto(pfx+"/profile")}
          class="btn option"
          >PROFILE
        </div>
        <div
          class:highlight={!systemTabCurrent}
          onclick={()=>goto(pfx+"/system")}
          class="btn option"
          >SYSTEM
        </div>
      {/if}
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
