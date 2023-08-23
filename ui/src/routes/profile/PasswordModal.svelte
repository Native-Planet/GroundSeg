<script>
	export let showModal
  import { modifyPassword } from '$lib/stores/websocket'
  import { createEventDispatcher } from 'svelte'
  import Fa from 'svelte-fa'
  import { faXmark } from '@fortawesome/free-solid-svg-icons';
  let dialog
	$: if (dialog && showModal) dialog.showModal();
  let cur = ''
  let pwd = ''
  let cfm = ''
  const dispatch = createEventDispatcher()
</script>

<dialog
	bind:this={dialog}
	on:close={() => (showModal = false)}
	on:click|self={() => dialog.close()}>
  <div class="x" on:click={()=>dispatch("close")}>
    <Fa icon={faXmark} size="1x"/>
  </div>
	<div on:click|stopPropagation>
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
</dialog>

<style>
	dialog {
    position: relative;
		border: none;
    border-radius: 16px;
		padding: 20px;
    background: var(--bg-base);
	}
	dialog::backdrop {
    background: rgba(92, 112, 96, 0.5);
	}
	dialog > div {
    width: calc(572px - 120px);
    font-family: var(--regular-font);
	}
  .x {
    position: absolute;
    background: var(--btn-secondary);
    background: var(--bg-modal);
    top: 0;
    right: 0;
    padding: 12px;
    border-radius: 0 16px 0 16px;
    width: 20px;
    text-align: center;
    cursor: pointer;
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

