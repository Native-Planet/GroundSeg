<script>
  import { scale, fly } from 'svelte/transition'
	import { quintOut } from 'svelte/easing'
  import { createEventDispatcher } from 'svelte'

  const dispatch = createEventDispatcher()

	import { api } from '$lib/api'
	import PrimaryButton from '$lib/PrimaryButton.svelte'
  
  import PierOptionsLogs from '$lib/PierOptionsLogs.svelte'
  import PierOptionsMinIO from '$lib/PierOptionsMinIO.svelte'
  import PierOptionsMeld from '$lib/PierOptionsMeld.svelte'
  import PierOptionsAdmin from '$lib/PierOptionsAdmin.svelte'

  export let remote, minIOReg, hasBucket, name, running, timeNow, frequency, meldHour, meldMinute, containers, expanded

  let selectedContainer = name

  // Available tabs
  let tabs = ['Logs','MinIO', 'Urbit'],
    activeTab = null,
    cur = null

  // Switch tabs
  // Todo: add transition
  const switchTab = (tab,i) => {
    if (expanded && (tab != 'Logs')) {toggleExpand()}

    if (activeTab == tab) {
      activeTab = null
    } else {
    activeTab = tab
    }

  }

  const toggleExpand = () => dispatch('toggleExpand')

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
  <div in:scale={{duration:120, delay: 200}}>
    <PierOptionsLogs on:toggleExpand={toggleExpand} {name} {containers} {expanded} />
  </div>
{/if}

{#if activeTab == 'MinIO'}
  <div in:scale={{duration:120, delay: 200}}>
    <PierOptionsMinIO {minIOReg} {remote} {hasBucket} {name}/>
  </div>
{/if}

{#if activeTab == 'Urbit'}
  <div class="main-wrapper" in:scale={{duration:120, delay: 200}}>
   <div class="admin-wrapper">
     <PierOptionsAdmin {name} {running} />
   </div>
   <div class="meld-wrapper">
     <PierOptionsMeld  {frequency} {timeNow} {running} {name} {meldHour} {meldMinute} />
   </div>
  </div>
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
  .main-wrapper {
    display: flex;
    text-align: center;
    padding-top: 20px;
    align-items: start;
  }
  .admin-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .meld-wrapper {
    flex:2;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

</style>
