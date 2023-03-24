<script>
	import { onMount, onDestroy } from 'svelte'

  import { scale } from 'svelte/transition'
	import { page } from '$app/stores'
	import { api, updateState, noconn } from '$lib/api'

	import Card from '$lib/Card.svelte'
  import Logo from '$lib/Logo.svelte'
  import ToggleAdvancedButton from '$lib/ToggleAdvancedButton.svelte'

	import PierHeader from '$lib/PierHeader.svelte'
	import PierProfile from '$lib/PierProfile.svelte'
	import PierCode from '$lib/PierCode.svelte'
	import PierUrl from '$lib/PierUrl.svelte'
  import PierMinIOSetup from '$lib/PierMinIOSetup.svelte'
  import PierMinIO from '$lib/PierMinIO.svelte'
	import PierNetwork from '$lib/PierNetwork.svelte'
  import PierOptions from '$lib/PierOptions.svelte'

	// load data into store
	export let data
	updateState(data)

	// default values
  let urbit
  let inView = true
  let loaded = false
  let code = null
  let advanced = false
  let failureCount = 0
  let isRunning = false

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
  }}


  const toggleAdvanced = () => {
    advanced = !advanced
  }

</script>

{#if inView && loaded}
<Card width="600px">

		<!-- Pier Header -->
		<PierHeader running={urbit.running} name={urbit.name}>
  		<Logo t="Urbit Ship Control Panel"/>
		</PierHeader>

		<!-- Pier Profile (public information) -->
    <div transition:scale={{duration:120, delay: 200}}>
      <PierProfile name={urbit.name} running={urbit.running} {code} />
    </div>

    {#if !advanced}
      <!-- Pier Credentials-->
      {#if (code != null) && (code.length == 27) && urbit.running}

        <!-- Landscape +code -->
        <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
          <PierCode code={code} />
        </div>
      {/if}

      <!-- Urbit Landscape URL -->
      {#if urbit.running}
        <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
          <PierUrl
            name={urbit.name}
            remote={urbit.remote}
            urbitUrl={urbit.urbitUrl}
            showUrbWeb={urbit.showUrbWeb}
            urbWebAlias={urbit.urbWebAlias}
          />
        </div>
      {/if}

        <!-- MinIO Console -->
        {#if urbit.wgReg && urbit.wgRunning}
          <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
            <PierMinIOSetup name={urbit.name} minIOReg={urbit.minIOReg} />
          </div>
          <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
            <PierMinIO minIOReg={urbit.minIOReg} minIOUrl={urbit.minIOUrl} />
          </div>
        {/if}

        <!-- Toggle Urbit Network -->
        <div in:scale={{duration:120, delay: 300}} out:scale={{duration:120}}>
          <PierNetwork name={urbit.name} remote={urbit.remote} wgReg={urbit.wgReg} wgRunning={urbit.wgRunning} />
        </div>
      <ToggleAdvancedButton on:click={toggleAdvanced} {advanced}/>
    {:else}
      <PierOptions
        remote={urbit.remote}
        minIOReg={urbit.minIOReg}
        hasBucket={urbit.hasBucket}
        name={urbit.name}
        running={urbit.running}
        timeNow={urbit.timeNow}
        frequency={urbit.frequency}
        meldHour={urbit.meldHour}
        meldMinute={urbit.meldMinute}
        containers={urbit.containers}
        meldOn={urbit.meldOn}
        meldLast={urbit.meldLast}
        meldNext={urbit.meldNext}
        autostart={urbit.autostart}
        loomSize={urbit.loomSize}
        wgReg={urbit.wgReg}
        urbWebAlias={urbit.urbWebAlias}
        s3WebAlias={urbit.s3WebAlias}
        on:click={toggleAdvanced}
      />
		{/if}
	</Card>
{/if}
