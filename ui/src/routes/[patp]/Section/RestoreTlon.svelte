<script>
  // import ToggleButton from '$lib/ToggleButton.svelte'
  // import FinalModal from './FinalModal.svelte';
  // import UnplugWarning from './UnplugWarning.svelte'
  // Style
  import "../theme.css"
  import { openModal } from 'svelte-modals'
  import { restoreTlonBackup } from '$lib/stores/websocket'

  import BackupsModal from './BackupsModal.svelte'

  export let patp
  import { URBIT_MODE } from '$lib/stores/data'

  const openBackupsModal = () => {
    openModal(BackupsModal, { isOpen: true, patp })
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
        on:click={openBackupsModal}>
        Show Backups
      </button>
    </div>
  </div>
</div>
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
