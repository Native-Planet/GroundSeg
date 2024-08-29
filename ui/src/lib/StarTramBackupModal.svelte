<script>
  import { startramBackupPassword } from '$lib/stores/websocket'
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'
  export let isOpen
  let newPassword = ''
</script>

{#if isOpen}
<Modal>
  <div class="wrapper">
    <!-- Info -->
    <h1>Setup Remote Backup Password</h1>
    <p>Encrypts your backups before sending them to StarTram.</p>

    <!-- Replacement Endpoint -->
    <h2>New Password</h2>
    <input placeholder="your-password" type="password" bind:value={newPassword} />

    <button
      disabled={(newPassword.length < 1)} 
      on:click={()=>{startramBackupPassword(newPassword); closeModal()}}
      >Set New Password
    </button>
  </div>
</Modal>
{/if}

<style>
  .wrapper {
    margin: 32px;
    font-family: var(--regular-font);
  }
  h1 {
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
  p {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
    width: 500px;
  }
  h2 {
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.2px;
  }
  input {
    flex: 1;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    border-radius: 16px;
    background: var(--Gray-100, #DDE3DF);
    padding: 16px 24px 18px 24px;
    width: calc(100% - 48px);
    border: none;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: .6;
    pointer-events: none;
  }
  button {
    margin-top: 56px;
    background-color: var(--btn-secondary);
    border-radius: 16px;
    cursor: pointer;
    padding: 0 48px;
    height: 65px;
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
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
