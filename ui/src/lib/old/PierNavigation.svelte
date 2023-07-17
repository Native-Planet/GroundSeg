<script>
  import { scale } from 'svelte/transition'
  import { createEventDispatcher } from 'svelte'
  export let activeTab

  const dispatch = createEventDispatcher()

  // Available tabs
  let tabs = ['Logs','Basic','Advanced']

</script>

  <!-- Navigation -->
  <div class="navbar" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
    {#each tabs as tab}
      <!-- Default Tab -->
      <div 
        class="tab" 
        on:click={()=>dispatch('click',tab)}
        class:active={tab == activeTab}
        transition:scale={{duration:120, delay: 200}}
        >
        {tab}
      </div>
    {/each}
  </div>
<style>
  .navbar {
    display: flex;
    margin: auto;
    gap: 6px;
    max-width: 600px;
    min-width: 360px;
    margin-bottom: 12px;
  }
  .tab {
    flex: 1;
    font-size: 14px;
    padding: 6px;
    text-align: center;
    border-radius: 8px;
    border: solid 1px #FFFFFF4D;
    cursor: pointer;
  }
  .tab:hover {background: #FFFFFF4D;}
  .active {
    background: var(--action-color);
    border-color: var(--action-color);
  }
  .active:hover {
    background: var(--action-color);
    opacity: .8;
  }
</style>
