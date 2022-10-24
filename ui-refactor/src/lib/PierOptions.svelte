<script>
  import { scale } from 'svelte/transition'
	import { api } from '$lib/api'
	import PrimaryButton from '$lib/PrimaryButton.svelte'
  
  import PierOptionsLogs from '$lib/PierOptionsLogs.svelte'
  import PierOptionsMinIO from '$lib/PierOptionsMinIO.svelte'
  import PierOptionsUrbit from '$lib/PierOptionsUrbit.svelte'

  export let remote, minIOReg, hasBucket, name, running

  // Available tabs
  let tabs = ['Logs','MinIO', 'Urbit'],
    activeTab = null,
    cur = null

  // Switch tabs
  // Todo: add transition
  const switchTab = (tab,i) => {
    if (activeTab == tab) {
      activeTab = null
    } else {
    activeTab = tab
    }
  }

</script>

<!-- Advanced Options Navigation -->
<div class="navbar">
  {#each tabs as tab, i}
    <!-- Check if tab is MinIO -->
    {#if tab == 'MinIO'}
      {#if !hasBucket || (minIOReg && remote)}
        <div 
          class="tab" 
          on:click={()=>switchTab(tab,i)}
          class:active={tab == activeTab}
          transition:scale={{duration:120, delay: 200}}
          >
          {tab}
        </div>
      {/if}
    <!-- Default Tab -->
    {:else}
      <div 
        class="tab" 
        on:click={()=>switchTab(tab,i)}
        class:active={tab == activeTab}
        transition:scale={{duration:120, delay: 200}}
        >
        {tab}
      </div>
    {/if}
  {/each}
</div>

<!-- Tab Contents -->
{#if activeTab == 'Logs'}
  <PierOptionsLogs />
{/if}
{#if activeTab == 'MinIO'}
  <PierOptionsMinIO {minIOReg} {remote} {hasBucket} {name}/>
{/if}
{#if activeTab == 'Urbit'}
  <PierOptionsUrbit {name} {running} />
{/if}

<style>
  .navbar {
    display: flex;
    margin-top: 12px;
    gap: 6px;
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
