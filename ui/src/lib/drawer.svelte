<script>
  import { fly } from 'svelte/transition';
  import { quintOut } from 'svelte/easing';
  import Fa from 'svelte-fa'
  import { faXmark } from '@fortawesome/free-solid-svg-icons';
  import { closeModal } from 'svelte-modals'
  export let title
</script>
<div 
  role="dialog"
  class="modal"
  transition:fly={{ duration: 300, x: 0, y: 500, opacity: 1, easing: quintOut }}

  >
  <div class="contents">
    <div class="title">{title}</div>
    <div class="x" on:click={closeModal}>
      <Fa icon={faXmark} size="1x"/>
    </div>
    <slot />
  </div>
</div>

<style>
  .title {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    text-transform: uppercase;
    font-family: BPdotsUnicase;
    font-size: 48px;
    font-style: normal;
    font-weight: 700;
    line-height: 32px; /* 66.667% */
    letter-spacing: -2.88px;
    margin-bottom: 56px;
  }
  .modal {
    position: fixed;
    top: 0;
    bottom: 0;
    right: 0;
    left: 0;
    display: flex;
    justify-content: center;
    align-items: end;
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
    width: 100vw;
    border-radius: 16px 16px 0 0;
    position: relative;
    background: white;
    height: calc(850px - (48px * 2));
    max-height: 80vh;
    padding: 48px;
  }
</style>
