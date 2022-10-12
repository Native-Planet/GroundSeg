<script>
	import { api } from '$lib/api'
	import PrimaryButton from '$lib/PrimaryButton.svelte'
	import { createEventDispatcher } from 'svelte'

	export let nw_label, minio_registered, patp, hasBucket
	let minIOLink = 'standard',
		pierExport = 'standard',
		bucketExport = 'standard'

	// temporary

	const dispatch = createEventDispatcher();

  const updateMinIO = () => {
		minIOLink = 'loading'
    let u = $api + "/urbit/minio_endpoint"
    const f = new FormData()
    f.append('pier', patp)

    fetch(u, {method: 'POST',body: f})
      .then(r => r.json())
			.then(d => { if (d == 200) {
				minIOLink = 'success'
				setTimeout(()=>minIOLink='standard', 3000)
			} else {
				minIOLink = 'failure'
				setTimeout(()=>minIOLink='standard', 3000)
        }})}

  const ejectPier = () => {
    pierExport = 'loading'
    let u = $api + "/urbit/eject"
    const f = new FormData()
					f.append(patp, 'eject')

    fetch(u, {method: 'POST',body: f})
    .then(res => { return res.blob(); })
    .then(d => {
      pierExport = 'standard'
      var a = document.createElement("a")
      a.href = window.URL.createObjectURL(d)
      a.download = patp
      a.click()
    })}

  const ejectBucket = () => {
    bucketExport = 'loading'
    let u = $api + "/urbit/minio/eject"
    const f = new FormData()
					f.append('pier', patp)

    fetch(u, {method: 'POST',body: f})
    .then(res => { return res.blob(); })
    .then(d => {
      bucketExport = 'standard'
      var a = document.createElement("a")
      a.href = window.URL.createObjectURL(d)
      a.download = 'bucket_' + patp
      a.click()
    })}


  const exportLog = c => {
    const u = $api + "/settings/logs"
    const f = new FormData()
    f.append('logs', patp)
    fetch(u, {method: 'POST', body: f})
      .then(r => r.json()).then(d => {
          var element = document.createElement('a')
          element.setAttribute('href', 'data:text/plain;charset=utf-8,' + encodeURIComponent(d))
          element.setAttribute('download', 'logs_' + patp)
          element.style.display = 'none'
          document.body.appendChild(element)
          element.click()
          document.body.removeChild(element)
    })}

</script>

<!-- Logs -->
<div class="info">
	<div class="title">Logs</div>
  <div class="button-wrapper">
		<PrimaryButton
			standard="View Urbit Pier Logs"
			noMargin={true}
			on:click={()=>dispatch('toggleLogs')} />
		<PrimaryButton
			standard="Export Urbit Pier Logs" 
	 		noMargin={true}
			background='#FFFFFF4D'
		 	on:click={exportLog} />
  </div>
</div>

<!-- MinIO -->
{#if hasBucket || (minio_registered && (nw_label == 'Remote'))}
	<div class="info">
		<div class="title">MinIO</div>
  	<div class="button-wrapper">

    	{#if minio_registered && (nw_label == 'Remote')}
				<PrimaryButton
					noMargin={true}
					standard="Link to Urbit"
  	     	success="MinIO linked!"
    	  	failure="Something went wrong"
    			loading="Linking..."
					status={minIOLink}
	 				on:click={updateMinIO} />
			{/if}

			{#if hasBucket}
				<PrimaryButton
					noMargin={true}
					background="#FFFFFF4D"
					standard="Export Bucket"
    			loading="Compressing your files.."
			 		status={bucketExport}
					on:click={ejectBucket} />
			{/if}
	  </div>
	</div>
{/if}

<!-- Pier Management -->
<div class="info">
  <div class="title">Pier Management</div>
  <div class="button-wrapper">
		<PrimaryButton
			noMargin={true}
			background="orange"
			standard="Export Urbit Pier"
   		loading="Compressing your pier.."
			status={pierExport}
			on:click={ejectPier} />

		<PrimaryButton
			noMargin={true}
			background="red"
			standard="Delete Urbit Pier"
	 on:click={()=>dispatch('deletePier')} />
	</div>
</div>

<style>
  .info {
    margin-bottom: 12px;
  }
  .title {
    font-weight: 700;
    margin-bottom: 12px;
    text-align: left;
  }
	.button-wrapper {
		display: flex;
		gap: 12px;
	}
	
</style>
