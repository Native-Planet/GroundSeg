<script>
  import { createEventDispatcher } from 'svelte'
  import Sigil from '$lib/Sigil.svelte'
  export let name, deleting = false

  const dispatch = createEventDispatcher();

  const handleBack = () => dispatch('back', null)
  const handleDelete = () => {dispatch('delete', null); deleting = true}

</script>
<div class="warning-title">WARNING:</div>
<div class="warning">
  <div class="sigil">
    <Sigil patp={name} size="87px" rad="15px"/>
  </div>
  <div class="warning-text">
    <div class="warning-content">
      Permanently delete ~{name}?
    </div>
    <div class="warning-content">
      This action cannot be undone. If you would like to proceed, please select 'Delete Pier'.
    </div>
  </div>
</div>
<div class="buttons">
  <button class="cancel" on:click={handleBack} >Cancel</button>
  <button class="delete" on:click={handleDelete} >Delet{deleting ? "ing" : "e Pier"}</button>
</div>

<style>
  .warning-title {
    font-size: 12px;
    font-weight: 700;
    padding-bottom: 12px;
    text-align: left;
  }
  .warning {
    color: inherit;
    display: flex;
    gap: 18px;
    align-items: center;
  }
  .warning-text {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .warning-content {
    font-size: 13px;
    opacity: .9;
  }
  .buttons {
    display: flex;
    margin-top: 24px;
  }

  .cancel {
    padding: 12px 20px 12px 20px;
    color: inherit;
    text-align: center;
    font-weight: 500;
    border-radius: 8px;
    background: #FFFFFF4D; 
    font-size: 16px;
    border: none;
    cursor: pointer;
  }
  .delete {
    background: #f48399;
    padding: 12px 20px 12px 20px;
    font-weight: 500;
    color: inherit;
    border-radius: 8px;
    border: none;
    font-size: 16px;
    font-family: inherit;
    margin-left: auto;
    cursor: pointer;
  }

</style>
