<script>
  import Fa from 'svelte-fa'
  import { faXmark } from '@fortawesome/free-solid-svg-icons';
  import { closeModal } from 'svelte-modals'

  import { fade } from 'svelte/transition';
	import { quintOut } from 'svelte/easing';
  export let width = 600
</script>
<div role="dialog" class="modal">
  <div
    class="contents" style="width:{width}px"
    transition:fade={{ duration: 200, easing:quintOut }}
    >
    <div class="x" on:click={()=>{console.log("astr");closeModal()}}>
      <Fa icon={faXmark} size="1x"/>
    </div>
    <slot />
  </div>
</div>

<style>
  .modal {
    position: fixed;
    top: 0;
    bottom: 0;
    right: 0;
    left: 0;
    display: flex;
    justify-content: center;
    align-items: center;
  }
  .x {
    position: absolute;
    background: var(--bg-modal);
    color: var(--text-color);
    top: 0;
    right: 0;
    padding: 12px;
    border-radius: 0 16px 0 16px;
    width: 20px;
    text-align: center;
    cursor: pointer;
  }
  .x:hover {
    background: var(--btn-secondary);  
    color: var(--text-card-color);
  }
  .contents {
    border-radius: 16px;
    position: relative;
    background: var(--text-card-color);
  }
</style>
