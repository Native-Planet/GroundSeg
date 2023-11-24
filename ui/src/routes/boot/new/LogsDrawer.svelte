<script>
  import Fa from 'svelte-fa'
  import { faXmark } from '@fortawesome/free-solid-svg-icons';
  import { structure } from '$lib/stores/websocket'
  import Drawer from '$lib/Drawer.svelte'
  import LogArea from './LogArea.svelte'
  export let patp
  export let isOpen
  let type = "urbit"
</script>

{#if isOpen}
<Drawer title={patp+" Logs"}>
  <LogArea container={patp} />
</Drawer>
{/if}


<style>
  .log-options {
    display: flex;
    gap: 32px;
  }
  .log-option {
    color: var(--Gray-200, #ABBAAE);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: -1.44px;
  }
  .active {
    color: var(--text-color);
  }
</style>
