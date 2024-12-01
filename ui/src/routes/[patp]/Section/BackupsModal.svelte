<script>
  import { structure } from '$lib/stores/data'
  import { restoreTlonBackup } from '$lib/stores/websocket'
  import BackupItem from './RestoreTlon/BackupItem.svelte'
  import Modal from '$lib/Modal.svelte'
  export let isOpen
  export let patp

  let remote = true

  let isSure = {bakType: null, timestamp: null}

  $: ship = ($structure?.urbits?.[patp]?.info) || {}
  $: remoteTlonBackups = (ship?.remoteTlonBackups) || []
  $: localDailyTlonBackups = (ship?.localDailyTlonBackups) || []
  $: localWeeklyTlonBackups = (ship?.localWeeklyTlonBackups) || []
  $: localMonthlyTlonBackups = (ship?.localMonthlyTlonBackups) || []
  $: tHandleRestoreTlonBackup = ($structure?.urbits?.[patp]?.transition?.handleRestoreTlonBackup) || ""

  const resetIsSure = () => {
    isSure = {bakType: null, timestamp: null}
  }

  const restoreLocalBackup = (backup, bakType) => {
    if (isSure.timestamp === backup.timestamp && isSure.bakType === bakType) {
      restoreTlonBackup(patp, false, backup.timestamp, backup.md5, bakType)
    } else {
      isSure = {bakType: bakType, timestamp: backup.timestamp}
    }
  }

  const restoreRemoteBackup = (backup) => {
    if (isSure.timestamp === backup.timestamp) {
      restoreTlonBackup(patp, true, backup.timestamp, backup.md5, "remote")
    } else {
      isSure = {bakType: "remote", timestamp: backup.timestamp}
    }
  }

  const toggleRemote = () => {
    remote = !remote
    isSure = {bakType: null, timestamp: null}
  }

</script>

<Modal>
  {#if isOpen}
  <div class="wrapper">
    <div class="text">Saved Backups</div>
    <div class="nav-wrapper">
      <div class="nav-indicator" class:left={remote} class:right={!remote}></div>
      <button class="nav-item" class:active={remote} on:click={toggleRemote}>Remote</button>
      <button class="nav-item" class:active={!remote} on:click={toggleRemote}>Local</button>
    </div>
    <div class="backups-wrapper">
    {#if remote}
      {#each remoteTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
        <BackupItem bakType="remote" {backup} {isSure} {tHandleRestoreTlonBackup} on:cancel={resetIsSure} on:restore={()=>restoreRemoteBackup(backup)}/>
      {/each}
    {:else}
      <div>Daily</div>
      {#each localDailyTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
        <BackupItem bakType="daily" {backup} {isSure} {tHandleRestoreTlonBackup} on:cancel={resetIsSure} on:restore={()=>restoreLocalBackup(backup, "daily")}/>
      {/each}
      <div>Weekly</div>
      {#each localWeeklyTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
        <BackupItem bakType="weekly" {backup} {isSure} {tHandleRestoreTlonBackup} on:cancel={resetIsSure} on:restore={()=>restoreLocalBackup(backup, "weekly")}/>
      {/each}
      <div>Monthly</div>
      {#each localMonthlyTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
        <BackupItem bakType="monthly" {backup} {isSure} {tHandleRestoreTlonBackup} on:cancel={resetIsSure} on:restore={()=>restoreLocalBackup(backup, "monthly")}/>
      {/each}
    {/if}
    </div>
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
  .backups-wrapper {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  button {
    display: inline-flex;
    padding: 16px 48px;
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
    margin-bottom: 32px;
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
