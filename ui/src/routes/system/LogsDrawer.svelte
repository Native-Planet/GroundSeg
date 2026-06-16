<script>
  import { structure } from '$lib/stores/data'
  import Drawer from '$lib/Drawer.svelte'
  import LogArea from '$lib/LogArea.svelte'
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
    font-family: var(--title-font);
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 48px; /* 200% */
    letter-spacing: 0;
    cursor: pointer;
  }
  .active {
    color: var(--text-color);
  }
</style>
