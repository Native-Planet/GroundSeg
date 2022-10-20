<script>
	import { onMount, onDestroy } from 'svelte'
	import { page } from '$app/stores'
	import { api, updateState } from '$lib/api'

	import Card from '$lib/Card.svelte'
  import SettingsButton from '$lib/SettingsButton.svelte'
  import Logo from '$lib/Logo.svelte'

	import PierHeader from '$lib/PierHeader.svelte'
	import PierProfile from '$lib/PierProfile.svelte'
	import PierCredentials from '$lib/PierCredentials.svelte'

	// load data into store
	export let data
	updateState(data)

	// default values
	let inView = false, loaded = false, urbit

	// start api loop
	onMount(()=> {
		inView = !inView
		update()
	})
	// stop api loop
	onDestroy(()=> inView = !inView)

	// api call
  const update = () => {
    if (inView) {
			fetch($api + '/urbit?urbit_id=' + $page.params.patp)
			.then(raw => raw.json())
			.then(res => { urbit = res; loaded = true})
			.catch(err => console.log(err))

			setTimeout(update, 1000)
	}}

	// temp
	let code = 'aaaaaa-aaaaaa-aaaaaa-aaaaaa'
</script>

<SettingsButton />

{#if inView && loaded}
<Card width="480px">

		<!-- Pier Header -->
		<PierHeader running={urbit.running} name={urbit.name}>
  		<Logo t="Urbit Ship Control Panel"/>
		</PierHeader>

		<!-- Pier Profile (public information) -->
		<PierProfile name={urbit.name} running={urbit.running}/>

	  <!-- Pier Credentials -->
	  {#if (code.length == 27) && urbit.running}
			<PierCredentials
      	name={urbit.name}
			  remote={urbit.remote}
				code={code}
				urbitUrl={urbit.urbitUrl}
			  wgReg={urbit.wgReg}
				wgRunning={urbit.wgRunning} />
		{/if}
		<!--
    		minIO={urbit.s3Url}
	      minIO_reg={data.pier.minio_registered}

		<!-- Advanced Options -->
		<!--
	  <div class="commands">
		  <div class="advanced" on:click={()=> advanced = !advanced}>
    		Advanced Options
      	<Fa icon={advanced ? faChevronUp : faChevronDown} size="0.8x" />
			</div>

			{#if advanced}
				<PierOptions 
					nw_label={data.nw_label}
					minio_registered={data.pier.minio_registered}
					patp={data.pier.name}
					hasBucket={data.hasBucket}
					on:toggleLogs={toggleLogs}
					on:exportLogs={()=>console.log("export")}
					on:deletePier={()=>deleteCheck=!deleteCheck}/>
			{/if}
		</div>
		-->

	</Card>
{/if}
