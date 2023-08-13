<script>
  import { onMount, afterUpdate } from 'svelte'
  import { structure, deleteUrbitShip } from '$lib/stores/websocket'
  import { showDeleteModal } from './store'

  export let patp

  $: transition = ($structure?.urbits?.[patp]?.transition) || {}
  $: tDeleteShip = (transition?.deleteShip) || null
</script>

<div class="wrapper">
  <div class="modal">
    <div class="warning">
      <div>You are attempting to DELETE all data related to:</div>
      <div> ~{patp}.</div>
      <div>This action cannot be undone.</div>
    </div>
    <div class="name">Please export data you want to save</div>

    <div class="export">
      <button class="btn-export">Export Urbit Pier</button>
      <button class="btn-export">Export MinIO Bucket</button>
    </div>

    <div class="check">
      <div class="checkbox"></div>
      <div class="check-text">I understand that this action cannot be undone.</div>
    </div>

    <div class="buttons">
      <button
        class="btn-cancel"
        on:click={()=>showDeleteModal.set(false)}
        >Back
      </button>
      <button
        on:click={()=>deleteUrbitShip(patp)}
        class="btn-delete"
        >
        {#if tDeleteShip}
          {tDeleteShip}
        {:else}
          Delete ~{patp}
        {/if}
      </button>
    </div>

  </div>
</div>

<style>
  .wrapper {
    position:absolute;
    left: 0;
    top: 0;
    backdrop-filter: blur(4px);
    -moz-backdrop-filter: blur(4px);
    -o-backdrop-filter: blur(2px);
    -webkit-backdrop-filter: blur(4px);
    width: 100vw;
    height: 100vh;
    background: #FFFFFF3D;
  }
  .modal {
    display: flex;
    flex-direction: column;
    position: absolute;
    top: calc(50vh - (392px/2));
    left: calc(50vw - (572px/2));
    background: var(--bg-modal);
    width: calc(572px - 80px);
    height: calc(392px - 80px);
    border-radius: 16px;
    padding: 40px;
    gap: 12px;
  }
  .header {
    font-family: var(--regular-font);
    font-size: 14px;
  }
  .regions {
    display: flex;
    gap: 20px;
  }
  .region {
    font-size: 12px;
    font-family: var(--regular-font);
    color: var(--text-color);
    border: solid 2px var(--btn-secondary);
    border-radius: 12px;
    padding: 8px 0;
    text-align: center;
    flex: 1;
    cursor: pointer;
  }
  .highlight {
    color: var(--text-card-color);
    background-color: var(--btn-secondary);
  }
  .name {
    font-family: var(--regular-font);
    font-size: 12px;
    margin-top: 12px;
  }
  .btn-delete {
    background: darkred;
    padding: 0 20px;
    color: var(--text-card-color);
    border-radius: 12px;
  }
  .btn-cancel {
    background: var(--btn-secondary);
    padding: 8px 20px;
    color: var(--text-card-color);
    border-radius: 12px;
  }
  .btn-delete:disabled {
    background: var(--btn-secondary);
    color: var(--text-color);
    opacity: .6;
  }
  .buttons {
    margin-top: 30px;
    display: flex;
    height: 36px;
    gap: 20px;
  }
  button:hover {
    cursor: pointer;
  }
  button {
    font-family: var(--regular-font);
    flex: 1;
  }
  .endpoint {
    font-family: var(--regular-font);
    font-size: 12px;
    width: calc(100% - 24px);
    line-height: 36px;
    border: solid 2px var(--btn-secondary);
    border-radius: 12px;
    background: none;
    padding-left: 20px;
  }
  input {
    width: calc(100% - 24px);
    line-height: 36px;
    border: solid 2px var(--btn-secondary);
    border-radius: 12px;
    background: none;
    padding-left: 20px;
  }
  input:active, :focus {
    outline: none; 
  }
  .get {
    font-family: var(--title-font);
    position: absolute;
    bottom: 20px;
    color: var(--text-color);
    font-size: 14px;
    text-decoration: underline;
  }
  .spacer {
    flex: 1;
  }
  .warning {
    font-size: 14px;
    text-align: center;
    background-color: var(--bg-warning);
    padding: 20px;
    border-radius: 12px;
  }
</style>
