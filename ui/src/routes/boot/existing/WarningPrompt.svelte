<script>
  import { goto } from '$app/navigation';
  import { createEventDispatcher } from 'svelte';
  export let status
  const dispatch = createEventDispatcher()
</script>
<div class="wrapper">
  <div class="modal">
    <div class="question">Is your pier offline?</div>
    <div class="info">Confirm that your pier is offline before importing your pier. It will be corrupted if it is online somewhere else.</div>
    <div class="buttons">
      <button
        class="no"
        on:click={()=>goto("/boot")}
        >No, it is not
      </button>
      <button
        class="yes"
        disabled={status != "uploading"}
        on:click={()=>dispatch('confirm')}
        >Yes, it is offline
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
    gap: 40px;
  }
  .question {
    font-size: 24px;
    line-height: 24px;
  }
  .info {
    font-size: 16px;
    flex: 1;
  }
  .buttons {
    display: flex;
    gap: 24px;
  }
  button {
    flex:1;
    font-size: 16px;
    line-height: 42px;
    color: var(--text-card-color);
    border-radius: 16px;
    cursor: pointer;
  }
  button:disabled {
    opacity: .6;
    pointer-events: none;
  }
  .no {
    background-color: var(--btn-secondary);
  }
  .yes {
    background-color: var(--bg-card);
  }
</style>
