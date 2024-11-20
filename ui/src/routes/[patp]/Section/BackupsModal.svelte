<script>
  import { structure } from '$lib/stores/data'
  import BackupItem from './RestoreTlon/BackupItem.svelte'
  import Modal from '$lib/Modal.svelte'
  export let isOpen
  export let patp

  let remote = true

  let isSure = undefined

  $: ship = ($structure?.urbits?.[patp]?.info) || {}
  $: remoteTlonBackups = (ship?.remoteTlonBackups) || []
  $: localTlonBackups = (ship?.localTlonBackups) || []
  $: tHandleRestoreTlonBackup = ($structure?.urbits?.[patp]?.transition?.handleRestoreTlonBackup) || ""

  const restoreLocalBackup = (backup) => {
    if (isSure === backup.timestamp) {
      restoreTlonBackup(patp, false, backup.timestamp, backup.md5)
    } else {
      isSure = backup.timestamp
    }
  }

  const restoreRemoteBackup = (backup) => {
    if (isSure === backup.timestamp) {
      restoreTlonBackup(patp, true, backup.timestamp, backup.md5)
    } else {
      isSure = backup.timestamp
    }
  }

</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="text">Saved Backups</div>
    <div class="nav-wrapper">
      <div class="nav-indicator" class:left={remote} class:right={!remote}></div>
      <button class="nav-item" class:active={remote} on:click={()=>remote = true}>Remote</button>
      <button class="nav-item" class:active={!remote} on:click={()=>remote = false}>Local</button>
    </div>
    {#if remote}
      {#each remoteTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
        <BackupItem {backup} {isSure} {tHandleRestoreTlonBackup} on:cancel={()=>isSure = undefined} on:restore={()=>restoreRemoteBackup(backup)}/>
      {/each}
    {:else}
      {#each localTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
        <BackupItem {backup} {isSure} {tHandleRestoreTlonBackup} on:cancel={()=>isSure = undefined} on:restore={()=>restoreLocalBackup(backup)}/>
      {/each}
    {/if}
  </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .text {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .sub {
    line-height: 32px;
    font-size: 20px;
    font-weight: 500;
    margin-top: 32px;
    color: var(--text-color);
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: #000;
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
  .nav-wrapper {
    display: flex;
    gap: 16px;
    background: var(--bg-base);
    border-radius: 16px;
    position: relative;
  }
  .nav-item {
    background: transparent;
    color: var(--text-color);
    flex: 1;
    z-index: 1;
  }
  .nav-indicator {
    position: absolute;
    bottom: 0;
    width: 50%;
    height: 100%;
    background: var(--btn-secondary);
    border-radius: 16px;
    z-index: 0;
  }
  .left {
    left: 0;
  }
  .right {
    right: 0;
  }
  .active {
    color: white;
  }
</style>
