<script>
  // import ToggleButton from '$lib/ToggleButton.svelte'
  // import { openModal } from 'svelte-modals'
  // import FinalModal from './FinalModal.svelte';
  // import UnplugWarning from './UnplugWarning.svelte'
  // Style
  import "../theme.css"
  import { restoreTlonBackup } from '$lib/stores/websocket'

  export let patp
  export let remoteTlonBackups
  export let localTlonBackups
  import { URBIT_MODE } from '$lib/stores/data'

  let showBackups = false

  const restoreLocalBackup = (backup) => {
    console.log("local", backup)
    restoreTlonBackup(patp, false, backup.timestamp, backup.md5)
  }

  const restoreRemoteBackup = (backup) => {
    console.log("remote", backup)
    restoreTlonBackup(patp, true, backup.timestamp, backup.md5)
  }
</script>

{#if !$URBIT_MODE}
<div class="section">
  <div class="section-left">
    <div class="section-title">Restore Tlon from Backup</div>
  </div>
  <div class="section-right">
    <div class="wrapper">
      <button
        class="btn domain-btn"
        class:active={showBackups}
        on:click={()=>showBackups = !showBackups}>
        {showBackups ? "Hide" : "Show"} Backups
      </button>
    </div>
  </div>
</div>
{#if showBackups}
<div class="backups-wrapper">
  <div class="backups-section">
    <div class="backups-title">Remote Backups</div>
    {#each remoteTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
      <button class="btn backup-item" on:click={() => restoreRemoteBackup(backup)}>
        {new Date(backup.timestamp * 1000).toLocaleString('en-US', {
          day: 'numeric',
          month: 'long',
          year: 'numeric',
          hour: 'numeric',
          minute: 'numeric',
          hour12: true
        })}
      </button>
    {/each}
  </div>
  <div class="backups-section">
    <div class="backups-title">Local Backups</div>
    {#each localTlonBackups.slice().sort((a, b) => b.timestamp - a.timestamp) as backup}
      <button class="btn backup-item" on:click={() => restoreLocalBackup(backup)}>
        {new Date(backup.timestamp * 1000).toLocaleString('en-US', {
          day: 'numeric',
          month: 'long',
          year: 'numeric',
          hour: 'numeric',
          minute: 'numeric',
          hour12: true
        })}
      </button>
    {/each}
  </div>
</div>
{/if}
{/if}

<style>
  .wrapper {
    display: flex;
    gap: 16px;
    margin: 16px 0;
    justify-content: flex-end;
  }
  .btn {
    color: #161D17;
    text-align: right;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 24px; /* 100% */
    height: 64px;
    letter-spacing: -1.44px;
    padding: 16px;
    display: flex;
    align-items: center;
    gap: 8px;
    border-radius: 12px;
    background: var(--NP_White, #F8F8F6);
    cursor: pointer;
  }
  .btn:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .domain-btn {
    background: #2C3A2E;
    color: white;
    padding: 0 48px;
  }
  .backups-wrapper {
    display: flex;
    gap: 64px;
    justify-content: center;
  }
  .backups-section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .backups-title {
    font-family: var(--regular-font);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    text-align: center;
}
</style>
