<script>
  import { structure } from '$lib/stores/data'
  import { shutdownDevice, restartDevice } from '$lib/stores/websocket'
  import { closeModal } from 'svelte-modals'
  import Modal from '$lib/Modal.svelte'

  export let isOpen

  const handleButton = () => {
    console.log("handled")
  }

  $: blockDevices = $structure?.system?.info?.blockDevices || {}
</script>

<Modal>
  <div class="wrapper">
      <div class="header">Disk Management</div>
      {#each blockDevices as d}
        <div class="name">Name: {JSON.stringify(d.name)}</div>
        <div class="name">Type: {JSON.stringify(d.type)}</div>
        <div class="name">Mountpoints: {JSON.stringify(d.mountpoints)}</div>
        <div class="name">Removable: {JSON.stringify(d.rm)}</div>
        {#each d.children as child}
          <div>{JSON.stringify(child)}</div>
        {/each}
      {/each}
    <button on:click={handleButton}>Button</button>
  </div>
</Modal>

<style>
  .wrapper {
    padding: 32px;
  }
  .header {
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
  .name {
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    max-width: 365px;
  }
  button {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: #000;
    border-radius: 16px;
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
    cursor: pointer;
  }
</style>
