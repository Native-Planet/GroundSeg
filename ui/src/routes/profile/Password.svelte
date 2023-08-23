<script>
  import { structure } from '$lib/stores/websocket'
  import PasswordModal from './PasswordModal.svelte'
  export let wide
  let showModal = false
</script>

<PasswordModal bind:showModal />

<div class="container {wide ? "wide" : "slim"}">
  <div class="title">COMPUTER</div>
  <div class="label">Password</div>
  <div class="wrapper">
    <div class="pwd">**************************</div>
    <button class="edit" on:click={()=>showModal=!showModal}>Edit</button>
  </div>
</div>

<style>
  .container {
    margin: auto;
  }
  .wrapper {
    display: flex;
    align-items: end;
    gap: 24px;
  }
  .pwd {
    width: calc(100% - 20px);
    font-family: var(--regular-font);
    color: var(--text-color);
    padding-left: 20px;
    border: 2px solid var(--btn-secondary);
    background-color: var(--bg-modal);
    border-radius: 12px;
    font-size: 16px;
    line-height: 36px;
  }
  .pwd:focus {
    outline: none;
  }
  .label {
    font-size: 12px;
    color: var(--text-color);
    margin-bottom: 8px;
  }
  .edit {
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
  }
  .wide {
    width: 992px;
    max-width: 98vw;
    padding: 56px;
  }
  .slim {
    width: 100vw;
  }
  .title {
    margin-bottom: 24px;
  }
</style>
