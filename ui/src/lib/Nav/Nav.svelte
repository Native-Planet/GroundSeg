<script>
  /*
  import { structure } from '$lib/stores/websocket'

  // Temp dev mode
  import DevToggle from '$lib/DevToggle.svelte'

  $: registered = ($structure?.profile?.startram?.info?.registered) || false
  $: running = ($structure?.profile?.startram?.info?.running) || false

  const handleBack = () => {
    const bootExist = $page.route.id.includes("new")
    const bootNew = $page.route.id.includes("existing")
    if (bootExist || bootNew) {
      goto("/boot")
    } else {
      goto("/")
    }
  }

<!--
<div class="wrapper {wide ? "wide" : "slim"}">
  <div class="app">
    GROUNDSEG {$version} - STARTRAM {
      !registered ? "UNREGISTERED" :
      running ? "ONLINE" : "OFFLINE"
      }
    <DevToggle />
  </div>
  {#if ($page.route.id == '/[patp]') || ($page.route.id.includes('/boot'))}
    <div class="back" on:click={handleBack}>
    </div>
  {:else}
    <div class="nav">
      <div class="ships">
        <div
          class:highlight={$page.route.id != "/(home)"}
          on:click={()=>goto("/")}
          class="btn option"
          >SHIPS
        </div>
      </div>
      <div
        class:highlight={$page.route.id != "/apps"}
        on:click={()=>goto("/apps")}
        class="btn option"
        >APPS
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
      -->



<!--
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
-->
  */
  import { goto } from '$app/navigation';
  import { page } from '$app/stores'
  import Desktop from './Desktop.svelte'
  import Mobile from './Mobile.svelte'
  import DesktopBack from './DesktopBack.svelte'
  import MobileBack from './MobileBack.svelte'

  // hardcoded tabs
  let tabs = ["startram","extensions","settings","ships"]

  // active tab
  $: active = handleActive($page.route.id)
  const handleActive = id => {
    if (id == "/(home)") {
      return "ships"
    }
    return id.substring(1)
  }

  // display nav
  $: show = handleShow($page.route.id)
  const handleShow = id => {
    for (let i = 0; i < tabs.length; i++) {
      let tab = tabs[i]
      if (tab == "ships") {
        tab = "(home)" 
      }
      if ("/"+tab == id) {
        return true
      }
    }
    return false
  }

  // when in back mode
  $: returnPosition = handleBack($page.route.id)
  const handleBack = id => {
    if (id.includes("boot/")) { // has subpath
      return "/boot" // back to boot menu
    }
    return "/"
  }


	$: innerWidth = 0
  $: isMobile = innerWidth < 960

  let URBIT_MODE = false

  const gotoPrefix = path => {
    let pfx = URBIT_MODE ? "/apps/groundseg" : ""
    goto(pfx+path)
  }

  const tabChange = e => {
    console.log(e)
    let tab = e.detail
    if (tab == "ships") {
      tab = ""
    }
    gotoPrefix("/"+tab)
  }

</script>
<svelte:window bind:innerWidth />

{#if show}
  <!-- always make sure its the correct width -->
  <div style="width:{innerWidth}px;">
    {#if isMobile}
      <Mobile {tabs} {active} on:tabbed={tabChange}/>
    {:else}
      <Desktop {tabs} {active} on:tabbed={tabChange}/>
    {/if}
  </div>
{/if}

<!-- back mode -->
{#if $page.route.id.includes("boot") || $page.route.id.includes("[patp]")}
    {#if isMobile}
      <MobileBack {returnPosition} on:tabbed={tabChange}/>
    {:else}
      <DesktopBack {returnPosition} on:tabbed={tabChange}/>
    {/if}
{/if}
