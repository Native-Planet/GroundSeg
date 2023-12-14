<script>
  import Fa from 'svelte-fa'
  import { faXmark } from '@fortawesome/free-solid-svg-icons';
  import { showLogs } from './store'
  import { structure } from '$lib/stores/data'
  import drawer from '$lib/drawer.svelte'
  import LogArea from './LogArea.svelte'
  export let title
  export let isOpen
  let type = "system"
  $: registered = ($structure?.profile?.startram?.info?.registered) || false
</script>

{#if isOpen}
<Drawer {title}>
  <div class="log-options">
    <div class="log-option" class:active={type=="system"} on:click={()=>type="system"}>GroundSeg</div>
    {#if registered}
      <div class="log-option" class:active={type=="startram"} on:click={()=>type="startram"}>StarTram</div>
    {/if}
  </div>
  {#if type == "startram"}
    <LogArea type="wireguard" />
  {:else}
    <LogArea type="system" />
  {/if}
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
