<script>
  import { onMount } from 'svelte'

  // Modal
  import Modal from '$lib/Modal.svelte'

  import { localBackup, scheduleLocalBackup } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  import BackupClock from './BackupClock.svelte'

  export let isOpen
  export let patp

  $: backupTime = ($structure?.urbits?.[patp]?.info?.backupTime)
  let newBackupTime = "unset"

  onMount(() => {
    newBackupTime = backupTime
  })

  const handleClockChange = e => {
    newBackupTime = e.detail
  }

  $: tLocalTlonBackup = ($structure?.urbits?.[patp]?.transition?.localTlonBackup) || ""
  $: tLocalTlonBackupSchedule = ($structure?.urbits?.[patp]?.transition?.localTlonBackupSchedule) || ""
</script>

<Modal width={720}>
  {#if isOpen}
    <div class="wrapper">
      <div class="header">Schedule Local Backups</div>
      <div class="micro">
        <div class="time-wrapper">
          <div class="micro-title">Time</div>
          <BackupClock on:select={handleClockChange} {patp} />
          <div class="button-wrapper">
            {#if tLocalTlonBackupSchedule == ""}
              <button
                disabled={backupTime == newBackupTime}
                on:click={()=>scheduleLocalBackup(patp,newBackupTime)}
              >
                Save Schedule
              </button>
            {:else if tLocalTlonBackupSchedule == "success"}
              <button disabled>Schedule Saved!</button>
            {:else if tLocalTlonBackupSchedule == "loading"}
              <button disabled>Saving Schedule...</button>
            {:else}
              <div style="flex: 1; flex-direction: column;">  
                <button class="now-btn" disabled>Error Saving Schedule</button>
                <div class="error-message">{tLocalTlonBackupSchedule}</div>
              </div>
            {/if}
            {#if tLocalTlonBackup == ""}
              <button class="now-btn" on:click={()=>localBackup(patp)}>Local Backup Now</button>
            {:else if tLocalTlonBackup == "loading"}
              <button class="now-btn" disabled>Backup in Progress...</button>
            {:else if tLocalTlonBackup == "success"}
              <button class="now-btn" disabled>Backup Completed!</button>
            {:else}
              <div style="flex: 1; flex-direction: column;">
                <button class="now-btn" disabled>Backup Failed</button>
                <div class="error-message">{tLocalTlonBackup}</div>
              </div>
            {/if}
          </div>
        </div>
      </div>
    </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .header {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .information {
    display: flex;
    gap: 32px;
  }
  .pack {
    height: 55px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    border: none;
    padding: 0 24px;
    display: flex;
    flex-direction: column;
    align-items: center; 
    justify-content: center;
    flex: 1;
  }
  .pack-title {
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 16px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .pack-subtitle {
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 12px;
    font-style: normal;
    font-weight: 500;
  }
  .macro {
    display: flex;
    gap: 16px;
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    max-width: 460px;
    margin: 32px 0;
    align-items: center;
  }
  .micro {
    display: flex;
    gap: 24px;
  }
  .time-wrapper {
    flex: 1;
  }
  .now-wrapper {
    flex: 1;
    margin-top: 32px;
  }
  .micro-title {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
    text-align: center;
  }
  input {
    width: 40px;
    text-align: center;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    max-width: 460px;
    line-height: 55px;
    min-width: 55px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    border: none;
  }
  /* Hide spinners in number input for Webkit browsers */
  input[type="number"]::-webkit-inner-spin-button,
  input[type="number"]::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }
  /* Hide spinners in number input for Firefox */
  input[type="number"] {
    -moz-appearance: textfield;
  }
  .select-wrapper {
    flex: 1;
  }
  .day-wrapper {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    justify-content: center;
  }
  .date-wrapper {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .day {
    user-select: none;
    flex-basis: 45%;
    cursor: pointer;
    padding: 16px 0;
    text-align: center;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
  }
  .date {
    flex-basis: 11%;
    padding: 4px 0;
    text-align: center;
    user-select: none;
    cursor: pointer;
    border: solid 1px var(--btn-secondary);
    border-radius: 4px;
  }
  .active {
    background: var(--btn-secondary);
    color: var(--text-card-color);
  }
  .button-wrapper {
    display: flex;
    gap: 32px;
  }
  .button-wrapper > button {
    flex: 1;
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: var(--btn-primary);
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
    height: 65px;
  }
  button:disabled {
    opacity: .6;
    pointer-events:none;
  }
  .stop {
    background: var(--btn-secondary);
  }
  .now-btn {
    background: black;
  }
  .error-message {
    font-family: var(--regular-font);
    color: red;
    text-align: center;
  }
</style>
