<script>
  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'
  import { setVereBits } from '$lib/stores/websocket'
  import { structure, URBIT_MODE } from '$lib/stores/data'

  export let isOpen
  export let patp
  export let currentBits = 32
  export let targetBits = 64
  export let ownShip = false

  $: loomSize = ($structure?.urbits?.[patp]?.info?.loomSize) || 31
  $: loomMB = (2**loomSize) / (1024*1024)
  $: running = ($structure?.urbits?.[patp]?.info?.running) || false

  const handleSwitch = () => {
    setVereBits(patp, targetBits)
    closeModal()
  }
</script>

<Modal width={640}>
  {#if isOpen}
    <div class="wrapper">
      <div class="header">Switch to {targetBits}-bit Vere</div>
      {#if $URBIT_MODE && ownShip}
        <div class="sub">You are currently accessing GroundSeg through this ship. You will temporarily lose access if you continue.</div>
      {/if}
      <div class="text">
        {patp} will be shut down{running ? "" : " if it is running"}, its snapshot will be migrated from the {currentBits}-bit to the {targetBits}-bit loom format, and it will then be started again. Large ships can take a while.
      </div>
      {#if targetBits == 64}
        <div class="text">
          Nouns use more memory on a 64-bit loom. Your loom is currently {loomMB} MB; consider raising it if RAM usage was already close to the limit.
        </div>
      {:else}
        <div class="text warn">
          The snapshot has to fit inside the 32-bit loom (16 GB at most). If it does not, the switch fails and the ship keeps running 64-bit Vere.
        </div>
      {/if}
      <div class="buttons">
        <button class="secondary" on:click={closeModal}>Cancel</button>
        <button on:click={handleSwitch}>Switch to {targetBits}-bit</button>
      </div>
    </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }
  .header {
    color: #000;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px;
    letter-spacing: -1.44px;
  }
  .sub {
    color: var(--text-color, #313933);
    font-family: Inter;
    font-size: 20px;
    font-weight: 500;
    line-height: 32px;
  }
  .text {
    color: var(--NP_Black, #313933);
    font-family: Inter;
    font-size: 20px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px;
    letter-spacing: -1.2px;
  }
  .warn {
    color: #a33d3d;
  }
  .buttons {
    display: flex;
    gap: 16px;
    justify-content: flex-end;
  }
  button {
    display: inline-flex;
    padding: 20px 40px;
    justify-content: center;
    align-items: center;
    background: black;
    border-radius: 16px;
    color: #FFF;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px;
    letter-spacing: -1.44px;
    cursor: pointer;
    border: 0;
  }
  button.secondary {
    background: var(--bg-base, #E8E4DB);
    color: var(--text-color, #313933);
  }
</style>
