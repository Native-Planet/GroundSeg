<script>
  import Fa from 'svelte-fa'
  import { faPlugCircleExclamation } from '@fortawesome/free-solid-svg-icons';
  import { URBIT_MODE } from '$lib/stores/data';
  import { openModal } from 'svelte-modals'
  import UnplugModal from './UnplugModal.svelte';
  export let component = ""
  export let ownShip = false
</script>
{#if $URBIT_MODE && ownShip}
  <div class="rad-wrapper">
    <div class="spacer"></div>
    <div class="rad" class:dead={component=="power"} on:click={()=>openModal(UnplugModal, {"component":component})}>
      <Fa icon={faPlugCircleExclamation} size="1.5x" />
    </div>
    <slot />
  </div>
{:else}
  <slot />
{/if}
<style>
  .rad-wrapper {
    display: flex;
    align-items: center;
    gap: 32px;
  }
  .spacer {
    flex: 1;
  }
  .rad {
    cursor: pointer;
    color: orange;
  }
  .dead {
    color: red;
  }
</style>
