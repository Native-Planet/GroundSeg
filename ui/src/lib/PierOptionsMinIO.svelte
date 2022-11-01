<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let minIOReg, remote, hasBucket, name

  // Button status
  let linkButtonStatus = 'standard',
    exportBucketStatus = 'standard'

	// Update Urbit S3 endpoint
	const updateMinIO = () => {
      linkButtonStatus = 'loading'
			fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'app':'pier','data':'s3-update'})
	  })
      .then(r => r.json())
			.then(d => { if (d == 200) {
				linkButtonStatus = 'success'
				setTimeout(()=>linkButtonStatus='standard', 3000)
			} else {
				linkButtonStatus = 'failure'
				setTimeout(()=>linkButtonStatus='standard', 3000)
        }})
      .catch(err => console.log(err))
  }

  const exportBucket = () => {
    exportBucketStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'app':'minio','data':'export'})
	  })
    .then(res => { return res.blob(); })
    .then(d => {
      exportBucketStatus = 'standard'
      var a = document.createElement("a")
      a.href = window.URL.createObjectURL(d)
      a.download = 'bucket_' + name
      a.click()
    })}

</script>

<div class="wrapper">
  {#if minIOReg && remote}
    <PrimaryButton
	    noMargin={true}
  		standard="Link MinIO to Urbit"
    	success="MinIO linked!"
      failure="Something went wrong"
      loading="Linking..."
	  	status={linkButtonStatus}
	 	  on:click={updateMinIO} />
  {/if}

{#if hasBucket}
  <PrimaryButton
		noMargin={true}
		background="#FFFFFF4D"
		standard="Export MinIO Bucket"
  	loading="Compressing your files.."
		status={exportBucketStatus}
		on:click={exportBucket} />
{/if}
</div>

<style>
  .wrapper {
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 12px;
    text-align: center;
  }
</style>
