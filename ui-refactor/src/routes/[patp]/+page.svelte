<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
	import { page } from '$app/stores'
	import { api, updateState } from '$lib/api'

  import SettingsButton from '$lib/SettingsButton.svelte'
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
  let inView = false,
    loaded = false,
    urbit, code = '',
    advanced = false

	// start api loop
	onMount(()=> {
		inView = !inView
    update()
    getUrbitCode()
  })

	// stop api loop
	onDestroy(()=> inView = !inView)

	// api call
  const update = () => {
    if (inView) {
			fetch($api + '/urbit?urbit_id=' + $page.params.patp)
			.then(raw => raw.json())
        .then(res => { handleData(res); loaded = true })
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

  const handleData = d => {
    if (d == 400) { window.location.href = "/" }
    if (d.name == $page.params.patp) { urbit = d }
  }

  const getUrbitCode = () => {
    if (inView) {
			fetch($api + '/urbit?urbit_id=' + $page.params.patp, {
			  method: 'POST',
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
  }}

</script>

<SettingsButton />

{#if inView && loaded}
<Card width="480px">

		<!-- Pier Header -->
		<PierHeader running={urbit.running} name={urbit.name}>
  		<Logo t="Urbit Ship Control Panel"/>
		</PierHeader>

		<!-- Pier Profile (public information) -->
    <PierProfile name={urbit.name} running={urbit.running} {code} />

	  <!-- Pier Credentials-->
	  {#if (code.length == 27) && urbit.running}

    	<!-- Landscape +code 
        placeholder code -->
      <div transition:scale={{duration:120, delay: 200}}>
    	  <PierCode code={code} />
      </div>

    	<!-- Urbit Landscape URL -->
      <div transition:scale={{duration:120, delay: 200}}>
    	  <PierUrl urbitUrl={urbit.urbitUrl} />
      </div>

    	<!-- MinIO Console -->
      {#if urbit.wgReg && urbit.wgRunning}
        <div transition:scale={{duration:120, delay: 200}}>
          <PierMinIOSetup name={urbit.name} minIOReg={urbit.minIOReg} />
        </div>
        <div transition:scale={{duration:120, delay: 200}}>
          <PierMinIO minIOReg={urbit.minIOReg} remote={urbit.remote} minIOUrl={urbit.minIOUrl} />
        </div>
      {/if}

      <!-- Toggle Urbit Network -->
      <div transition:scale={{duration:120, delay: 200}}>
		    <PierNetwork name={urbit.name} remote={urbit.remote} wgReg={urbit.wgReg} wgRunning={urbit.wgRunning} />
      </div>

    {/if}

    <!-- Advanced Options -->
    <ToggleAdvancedButton {advanced} on:click={()=> advanced = !advanced} />

		{#if advanced}
      <PierOptions
        remote={urbit.remote}
        minIOReg={urbit.minIOReg}
        hasBucket={urbit.hasBucket}
        name={urbit.name}
        running={urbit.running}
        />
		{/if}

	</Card>
{/if}
