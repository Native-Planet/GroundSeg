<script>
    import { structure } from '$lib/stores/data'
  import { 
    devStartramReminder,
    devStartramReminderToggle,
    devRemoteBackupTlon,
    printMounts,
    devBackupTlon,
    devRestoreTlon
  } from '$lib/stores/websocket'

  let openDevPanel = !false

  $: ships = ($structure?.urbits) || {}
  $: shipNames = Object.keys(ships)
</script>
{#if openDevPanel}
<div class="dev">
  <div class="close-panel">
    <div class="close-spacer"></div>
    <button class="close" on:click={()=>openDevPanel=false}>Close</button>
  </div>
  <div class="panel">
    <h1>Disk</h1>
    <div class="panel-buttons">
      <button on:click={printMounts}>Print Mounts</button>
    </div>
  </div>
  <div class="panel">
    <h1>Startram</h1>
    <div class="panel-buttons">
        <button on:click={devStartramReminder}>Remind Startram</button>
        <button on:click={()=>devStartramReminderToggle(true)}>Reminded</button>
        <button on:click={()=>devStartramReminderToggle(false)}>Have not Reminded</button>
    </div>
  </div>
  <div class="panel">
    <h1>Backup</h1>
    <div class="panel-buttons">
        <button on:click={devBackupTlon}>Local All Ships</button>
        <button on:click={devRemoteBackupTlon}>Remote All Ships</button>
    </div>
  </div>
  <div class="panel">
    <h1>Remote Restoration</h1>
    <div class="panel-buttons">
        {#each shipNames as patp}
            <button on:click={()=>devRestoreTlon(patp, true)}>{patp}</button>
        {/each}
    </div>
  </div>
  <div class="panel">
    <h1>Local Restoration</h1>
    <div class="panel-buttons">
        {#each shipNames as patp}
            <button on:click={()=>devRestoreTlon(patp, false)}>{patp}</button>
        {/each}
    </div>
  </div>
</div>
{:else}
  <button class="open" on:click={()=>openDevPanel=true}>Dev Panel</button>
{/if}

<style>
    h1 {
        font-size: 16px;
        font-weight: 400;
        font-family: var(--regular-font);
        color: white;
    }
  .close-panel {
    display: flex;
  }
  .close-spacer {
    flex: 1;
  }
  .open {
    background: black;
    color: white;
    border-radius: 16px;
    margin: 32px;
    position: fixed;
    bottom: 0;
    left: 0;
    padding: 16px;
  }
  .dev {
    border-radius: 16px;
    position: fixed;
    bottom: 32px;
    left: 32px;
    display: flex;
    flex-direction: column;
    align-items: left;
    justify-content: center;
    gap: 16px;
    background: black;
    padding: 32px;
    padding-top: 16px;
    border: solid 2px white;
    opacity: 0.8;
  }
  .panel-buttons {
    display: flex;
    gap: 8px;
  }
  button {
    padding: 6px;
    border: solid 1px white;
    color: white;
    border-radius: 8px;
    cursor: pointer;
  }
  button:hover {
    opacity: 0.6;
  }
</style>