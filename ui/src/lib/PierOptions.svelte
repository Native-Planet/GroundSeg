<script>
  import { scale, fly } from 'svelte/transition'
	import { quintOut } from 'svelte/easing'
  import { createEventDispatcher } from 'svelte'
  import ToggleAdvancedButton from '$lib/ToggleAdvancedButton.svelte'

  const dispatch = createEventDispatcher()

	import { api } from '$lib/api'
	import PrimaryButton from '$lib/PrimaryButton.svelte'
  
  import PierDeletionCheck from '$lib/PierDeletionCheck.svelte'
  import PierOptionsLogs from '$lib/PierOptionsLogs.svelte'

  import PierOptionsLoom from '$lib/PierOptionsLoom.svelte'
  import PierOptionsMinIO from '$lib/PierOptionsMinIO.svelte'
  import PierOptionsMeld from '$lib/PierOptionsMeld.svelte'
  import PierOptionsDomain from '$lib/PierOptionsDomain.svelte'
  import PierOptionsAdmin from '$lib/PierOptionsAdmin.svelte'

  export let remote
  export let minIOReg
  export let hasBucket
  export let name
  export let running
  export let timeNow
  export let frequency
  export let meldHour
  export let meldMinute
  export let meldOn
  export let meldLast
  export let meldNext
  export let containers
  export let autostart
  export let loomSize
  export let wgReg
  export let urbWebAlias
  export let s3WebAlias

  let activeTab = 'Settings'
  let cur = null
  let isPierDeletion = false
  let selectedContainer = name

  // Available tabs
  let tabs = ['Logs','Settings']

  // Switch tabs
  const switchTab = (tab) => {
    activeTab = tab
  }

</script>

{#if isPierDeletion}
  <div in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
    <PierDeletionCheck {name} {hasBucket} on:cancel={()=>isPierDeletion = false} /> 
  </div>
{:else}
  <!-- Advanced Options Navigation -->
  <div class="navbar" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
    {#each tabs as tab}
      <!-- Default Tab -->
      <div 
        class="tab" 
        on:click={()=>switchTab(tab)}
        class:active={tab == activeTab}
        transition:scale={{duration:120, delay: 200}}
        >
        {tab}
      </div>
    {/each}
  </div>

  <!-- Tab Contents -->
  {#if activeTab == 'Logs'}
    <div  in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
      <PierOptionsLogs {name} {containers} on:click={()=>dispatch('click')}/>
    </div>
  {/if}

  {#if activeTab == 'Settings'}
    <div class="main-wrapper" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
      <div class="left-wrapper">
        <PierOptionsMeld 
          {frequency}
          {timeNow}
          {running}
          {name}
          {meldHour}
          {meldMinute}
          {meldOn}
          {meldLast}
          {meldNext}
        />
        <PierOptionsLoom {name} {loomSize} />
        <PierOptionsAdmin
          {name}
          {isPierDeletion}
          {hasBucket}
          {autostart}
          on:delete={()=>isPierDeletion = true}
        />
      </div>
      <div class="right-wrapper">
        <PierOptionsMinIO {minIOReg} {remote} {hasBucket} {name}/>
        {#if wgReg}
          <PierOptionsDomain {name} alias={urbWebAlias} title="Urbit Ship Custom Domain" svcType="urbit-web" stdText="Submit ship domain">
            <div class="info" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
              Access your Urbit ship from a second domain. Please read
              <a href="https://www.nativeplanet.io/custom-startram-domains" target="_blank">this guide</a>
              for more information.
            </div>
          </PierOptionsDomain>
          <PierOptionsDomain {name} alias={s3WebAlias} title="MinIO Custom Domain" svcType="minio" stdText="Submit MinIO domain" >
            <div class="info" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
              Set your MinIO bucket's public URL to another domain. Please read
              <a href="https://www.nativeplanet.io/custom-startram-domains" target="_blank">this guide</a>
              for more information.
            </div>
          </PierOptionsDomain>
        {/if}
      </div>
    </div>
    <ToggleAdvancedButton on:click={()=>dispatch('click')} advanced={true} />
  {/if}
{/if}

<style>
  .navbar {
    display: flex;
    margin: auto;
    gap: 6px;
    width: 60%;
    min-width: 360px;
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
    gap: 12px;
  }
  .left-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .right-wrapper {
    flex:1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .info {
    font-size: 11px;
    padding: 8px 20px 8px 20px;
  }
  a {
    color: inherit;
    text-decoration: underline;
  }

</style>
