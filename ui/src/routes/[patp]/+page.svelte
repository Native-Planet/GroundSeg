<script>
	import { onMount, onDestroy } from 'svelte'

  import { scale } from 'svelte/transition'
	import { page } from '$app/stores'
	import { isPortrait, api, updateState, noconn } from '$lib/api'

	import Card from '$lib/Card.svelte'
  import Logo from '$lib/Logo.svelte'
  import ToggleAdvancedButton from '$lib/ToggleAdvancedButton.svelte'
	import PrimaryButton from '$lib/PrimaryButton.svelte'

	import PierHeader from '$lib/PierHeader.svelte'

  import PierNavigation from '$lib/PierNavigation.svelte'
  import PierOptionsLogs from '$lib/PierOptionsLogs.svelte'

	import PierProfile from '$lib/PierProfile.svelte'
	import PierCode from '$lib/PierCode.svelte'
	import PierUrl from '$lib/PierUrl.svelte'
  import PierMinIOSetup from '$lib/PierMinIOSetup.svelte'
  import PierMinIO from '$lib/PierMinIO.svelte'
	import PierNetwork from '$lib/PierNetwork.svelte'

  // Left Advanced
  import PierAdvancedCode from '$lib/PierAdvancedCode.svelte'
  import PierOptionsMeld from '$lib/PierOptionsMeld.svelte'
  import PierOptionsLoom from '$lib/PierOptionsLoom.svelte'

  // Center Advanced
  import PierAdvancedUrl from '$lib/PierAdvancedUrl.svelte'
  import PierAdvancedNetwork from '$lib/PierAdvancedNetwork.svelte'
  import PierOptionsAdmin from '$lib/PierOptionsAdmin.svelte'
  import PierOptionsMinIO from '$lib/PierOptionsMinIO.svelte'

  // Right Advanced
  import PierOptionsDomain from '$lib/PierOptionsDomain.svelte'

	// load data into store
	export let data
	updateState(data)

	// default values
  let urbit
  let inView = true
  let loaded = false
  let code = null
  let failureCount = 0
  let isRunning = false
  let isPierDeletion = false

  let activeTab = 'Basic' //TODO: set to last tab
  
  let advanced = (activeTab == "Advanced")

	// start api loop
	onMount(()=> {
    api.set("http://" + $page.url.hostname + ":27016")
    if (data['status'] == 404) {
      window.location.href = "/login"
    }

    if (data['status'] == 'setup') {
      window.location.href = "/setup"
    }

    update()
    getUrbitCode()
  })

	// stop api loop
	onDestroy(()=> inView = false)

	// api call
  const update = () => {
    if (inView && !$noconn) {
      fetch($api + '/urbit?urbit_id=' + $page.params.patp, {
        credentials: "include"
      })
			.then(raw => raw.json())
        .then(res => {
          if (res == 404) {
            window.location.href = "/login"
          }
          handleData(res)
        })
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

  const handleData = d => {
    if (d == 400) { 
      failureCount = ++failureCount

      if (failureCount > 3) {
        window.location.href = "/" }
    }

    if (d.name == $page.params.patp) { 
      loaded = true
      failureCount = 0
      urbit = d 
      isRunning = urbit.running
    }
  }

  const getUrbitCode = () => {
    if (inView) {
      if (isRunning) {
        fetch($api + '/urbit?urbit_id=' + $page.params.patp, {
          method: 'POST',
          credentials: "include",
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({'app':'pier','data':'+code'})
        })
        .then(r => r.json())
        .then(d => {
          code = d
          if (d.length == 27) {
            setTimeout(getUrbitCode, 1800000)
          } else {
            setTimeout(getUrbitCode, 1000)
          }
        })
      } else {
        setTimeout(getUrbitCode, 1000)
      }
      console.log(code)
  }}


  const toggleAdvanced = () => {
    advanced = !advanced
  }

  // Switch tabs
  const switchTab = e => {
    activeTab = e.detail
    advanced = activeTab == "Advanced"
  }
</script>

{#if inView && loaded}
  <Card width="{advanced ? 900 : 600}px" devMode={urbit.devMode}>

    <!-- Pier Header -->
    <PierHeader running={urbit.running} name={urbit.name}>
      <Logo t="Urbit Ship Control Panel"/>
    </PierHeader>

    <!-- Pier Profile (public information) -->
    <div transition:scale={{duration:120, delay: 200}}>
      <PierProfile
        {code}
        name={urbit.name}
        running={urbit.running}
        devMode={urbit.devMode}
        click={urbit.click}
      />
    </div>

    <!-- Navbar -->
    <PierNavigation on:click={switchTab} {activeTab} />

    <!-- Tab Contents -->
    {#if activeTab == 'Logs'}
      <div in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
        <PierOptionsLogs name={urbit.name} containers={urbit.containers} on:click={()=>console.log("export")}/>
      </div>
    {/if}

    {#if activeTab == 'Basic'}
      <!-- Landscape +code -->
      <div class:disabled={(code == null) || (code.length != 27) || !urbit.running} in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
        <PierCode code={code} />
      </div>

      <!-- Urbit Landscape URL -->
      <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
        <PierUrl
          name={urbit.name}
          remote={urbit.remote}
          urbitUrl={urbit.urbitUrl}
          showUrbWeb={urbit.showUrbWeb}
          urbWebAlias={urbit.urbWebAlias}
        />
      </div>

      <!-- MinIO Console -->
      {#if urbit.wgReg && urbit.wgRunning}
        {#if urbit.minIOReg}
          <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
            <PierMinIO minIOReg={urbit.minIOReg} minIOUrl={urbit.minIOUrl} />
          </div>
        {:else}
          <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
            <PierMinIOSetup name={urbit.name} minIOReg={urbit.minIOReg} />
          </div>
        {/if}
      {/if}

      <!-- Toggle Urbit Network -->
      <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
        <PierNetwork name={urbit.name} remote={urbit.remote} wgReg={urbit.wgReg} wgRunning={urbit.wgRunning} />
      </div>
    {/if}

    <!-- Advanced Options -->
    {#if activeTab == 'Advanced'}
      <!-- Three columns -->
      <div class="main-wrapper" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
        <!-- Left side -->
        <div class="left-wrapper">
          <PierAdvancedCode {code} disabled={(code == null) || (code.length != 27) || !urbit.running} />
          <PierOptionsMeld 
            disabled={urbit.devMode}
            timeNow={urbit.timeNow}
            frequency={urbit.frequency}
            name={urbit.name}
            running={urbit.running}
            meldHour={urbit.meldHour}
            meldMinute={urbit.meldMinute}
            meldOn={urbit.meldOn}
            meldLast={urbit.meldLast}
            meldNext={urbit.meldNext}
          />
          {#if $isPortrait}
            <PierOptionsLoom name={urbit.name} loomSize={urbit.loomSize} />
          {/if}
          <PierOptionsAdmin 
            name={urbit.name}
            devMode={urbit.devMode}
            click={urbit.click}
            autostart={urbit.autostart}
            on:delete={()=>isPierDeletion = true}
          />
        </div>
        <!-- Center -->
        {#if !$isPortrait}
          <div class="center-wrapper">
            <PierAdvancedUrl
              name={urbit.name}
              remote={urbit.remote}
              urbitUrl={urbit.urbitUrl}
              showUrbWeb={urbit.showUrbWeb}
              urbWebAlias={urbit.urbWebAlias}
            />
            <PierAdvancedNetwork
              name={urbit.name}
              remote={urbit.remote}
              wgReg={urbit.wgReg}
              wgRunning={urbit.wgRunning}
            />
            <PierOptionsLoom name={urbit.name} loomSize={urbit.loomSize} />
          </div>
        {/if}
        <!-- Right side -->
        <div class="right-wrapper">
          {#if $isPortrait}
            <PierAdvancedUrl
              name={urbit.name}
              remote={urbit.remote}
              urbitUrl={urbit.urbitUrl}
              showUrbWeb={urbit.showUrbWeb}
              urbWebAlias={urbit.urbWebAlias}
            />
            <PierAdvancedNetwork
              name={urbit.name}
              remote={urbit.remote}
              wgReg={urbit.wgReg}
              wgRunning={urbit.wgRunning}
            />
          {/if}
          <PierOptionsMinIO 
            remote={urbit.remote}
            minIOReg={urbit.minIOReg}
            hasBucket={urbit.hasBucket}
            disabled={urbit.devMode || !urbit.minIOReg}
          />
          <PierOptionsDomain
            name={urbit.name}
            disabled={!urbit.wgReg}
            alias={urbit.urbWebAlias}
            title="Urbit Ship Custom Domain"
            svcType="urbit-web"
            stdText="Submit ship domain"
            >
            <div class="info" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
              Access your Urbit ship from a second domain. Please read
              <a href="https://www.nativeplanet.io/custom-startram-domains" target="_blank">this guide</a>
              for more information.
            </div>
          </PierOptionsDomain>
          <PierOptionsDomain
            name={urbit.name}
            disabled={!urbit.wgReg || !urbit.minIOReg}
            alias={urbit.s3WebAlias}
            title="MinIO Custom Domain"
            svcType="minio"
            stdText="Submit MinIO domain"
            >
            <div class="info" in:scale={{duration:120, delay: 300}} out:scale={{duration:60, delay:0}}>
              Set your MinIO bucket's public URL to another domain. Please read
              <a href="https://www.nativeplanet.io/custom-startram-domains" target="_blank">this guide</a>
              for more information.
            </div>
          </PierOptionsDomain>
        </div>
      </div>
    {/if}
	</Card>
{/if}

<style>
  .disabled {
    color: #ff00004d;
    pointer-events: none;
    opacity: .6;
  }
  .main-wrapper {
    display: flex;
    text-align: center;
    align-items: start;
    gap: 12px;
  }
  .left-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .center-wrapper {
    flex:1;
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
