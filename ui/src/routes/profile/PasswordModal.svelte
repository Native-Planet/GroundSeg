<script>
  import { afterUpdate } from 'svelte'
  import { modifyPassword } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'

  export let isOpen

  let cur = ''
  let pwd = ''
  let cfm = ''

  /*
  afterUpdate(()=>{
    if (tRegister == "done") {
      closeModal()
    }
  })
  */
</script>

{#if isOpen}
  <Modal>
    <div class="wrapper">
      <div class="pwds">
        <div class="edit-title">Edit Password</div>
        <div class="pw-wrapper">
          <div class="label">Current Password</div>
          <input type="password" bind:value={cur} />
        </div>
        <div class="pw-wrapper">
          <div class="label">New Password</div>
          <input type="password" bind:value={pwd} />
        </div>
        <div class="pw-wrapper">
          <div class="label">Confirm Password</div>
          <input type="password" bind:value={cfm} />
        </div>
      </div>
      <button
        disabled={(cfm.length < 1) || (pwd != cfm)}
        on:click={()=>modifyPassword(cur,cfm)}
        >Save
      </button>
    </div>
  </Modal>
{/if}

<style>
  .wrapper {
    margin: 20px;
    font-family: var(--regular-font);
  }
  .pwds {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .edit-title {
    font-weight: 400;   
    margin-bottom: 20px;
  }
  .pw-wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .label {
    font-size: 14px;
    color: var(--btn-secondary);
    padding-left: 8px;
  }
  input {
    border: none;
    background: var(--bg-modal);
    padding: 10px 20px;
    border-radius: 12px;
  }
  input:focus {
    outline: none;
  }
  button {
    margin-top: 20px;
    font-family: var(--regular-font);
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 42px;
    font-size: 12px;
    border-radius: 12px;
    cursor: pointer;
    justify-content: center;
    align-items: center;
    padding: 0 48px;
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
</style>
