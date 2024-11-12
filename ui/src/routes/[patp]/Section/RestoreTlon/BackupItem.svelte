<script>
  import { createEventDispatcher } from 'svelte';
  import { structure } from '$lib/stores/data'

  export let backup;
  export let isSure;
  export let tHandleRestoreTlonBackup = ""
  const dispatch = createEventDispatcher();

</script>
    {#if isSure === backup.timestamp}
      <div class="backup-selected-wrapper">
        <div class="backup-selected">{new Date(backup.timestamp * 1000).toLocaleString('en-US', {
          day: 'numeric',
          month: 'long',
          year: 'numeric',
          hour: 'numeric',
          minute: 'numeric',
          hour12: true
        })}</div>
        <div class="backup-button-wrapper">
          <button class="backup-cancel" disabled={tHandleRestoreTlonBackup != ""} on:click={() => dispatch('cancel')}>
            Cancel
          </button>
          {#if tHandleRestoreTlonBackup == ""}
            <button class="backup-restore" on:click={() => dispatch('restore')}>Restore</button>
          {:else if tHandleRestoreTlonBackup == "loading"}
            <button class="backup-restore" disabled>Restoring...</button>
          {:else if tHandleRestoreTlonBackup == "success"}
            <button class="backup-restore" disabled>Restored!</button>
          {:else}
            <button class="backup-restore" disabled>Failed</button>
          {/if}
        </div>
      </div>
    {:else}
      <button disabled={isSure !== undefined} class="btn backup-item" on:click={() => dispatch('restore')}>
        {new Date(backup.timestamp * 1000).toLocaleString('en-US', {
          day: 'numeric',
          month: 'long',
          year: 'numeric',
          hour: 'numeric',
          minute: 'numeric',
          hour12: true
        })}
      </button>
    {/if}

<style>
.backup-selected-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-radius: 12px;
  border: 1px solid white;
  padding: 64px;
}
.backup-selected {
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
    margin-bottom: 32px;
}
.backup-button-wrapper {
  display: flex;
  gap: 8px;
}
.backup-restore {
  background: var(--btn-primary);
  color: var(--text-card-color);
  flex: 1;
  cursor: pointer;
  padding: 16px;
  border-radius: 16px;
}
.backup-cancel {
  color: var(--text-card-color);
  background: #FF6B6B;
  flex: 1;
  cursor: pointer;
  padding: 16px;
  border-radius: 16px;
}
.backup-restore:disabled {
  opacity: .6;
  pointer-events: none;
}
.backup-cancel:disabled {
  opacity: .6;
  pointer-events: none;
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
</style>