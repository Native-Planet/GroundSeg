<script>
  import { afterUpdate } from 'svelte'
  import { modifyPassword } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/modal.svelte'

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
    margin: 32px;
    font-family: var(--regular-font);
  }
  .pwds {
  }
  .edit-title {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .pw-wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .label {
    margin-top: 16px;
    color: var(--Gray-400, #5C7060);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 240% */
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
    border: none;
  }
  input:focus {
    outline: none;
  }
  button {
    margin-top: 20px;
    font-family: var(--regular-font);
    background: var(--btn-secondary);
    color: var(--text-card-color);
    height: 65px;
    font-size: 12px;
    border-radius: 16px;
    cursor: pointer;
    justify-content: center;
    align-items: center;
    padding: 0 48px;
    cursor: pointer;

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
    opacity: .6;
    pointer-events: none;
  }
</style>
