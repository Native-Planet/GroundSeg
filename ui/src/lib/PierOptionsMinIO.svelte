<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let minIOReg, remote, hasBucket, name

  // Button status
  let linkButtonStatus = 'standard',
    unlinkButtonStatus = 'standard',
    exportBucketStatus = 'standard'

	// Update Urbit S3 endpoint
	const updateMinIO = () => {
      linkButtonStatus = 'loading'
			fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
        credentials: "include",
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

	const unlinkMinIO = () => {
      unlinkButtonStatus = 'loading'
			fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
        credentials: "include",
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({'app':'pier','data':'s3-unlink'})
	  })
      .then(r => r.json())
			.then(d => { if (d == 200) {
				unlinkButtonStatus = 'success'
				setTimeout(()=>unlinkButtonStatus='standard', 3000)
			} else {
				unlinkButtonStatus = 'failure'
				setTimeout(()=>unlinkButtonStatus='standard', 3000)
        }})
      .catch(err => console.log(err))
  }

  const exportBucket = () => {
    exportBucketStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
			method: 'POST',
        credentials: "include",
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

<div class="bg">
  <div class="option-title">MinIO Settings</div>
  <div class="wrapper">
    <div class="top-wrapper">
      {#if minIOReg && remote}
        <PrimaryButton
          noMargin={true}
          standard="Link to Urbit"
          success="MinIO linked!"
          failure="Something went wrong"
          loading="Linking..."
          status={linkButtonStatus}
          on:click={updateMinIO} />
        <PrimaryButton
          noMargin={true}
          background="#FFFFFF4D"
          standard="Unlink"
          success="MinIO unlinked from Urbit!"
          failure="Something went wrong"
          loading="Removing link..."
          status={unlinkButtonStatus}
          on:click={unlinkMinIO} />
      {/if}
    </div>
    {#if hasBucket}
      <PrimaryButton
        noMargin={true}
        background="#FFFFFF4D"
        standard="Export Bucket"
        loading="Compressing your files.."
        status={exportBucketStatus}
        on:click={exportBucket} />
    {/if}
  </div>
</div>

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .option-title {
    font-size: 14px;
    color: inherit;
    margin-bottom: 12px;
  }
  .wrapper {
    display: flex;
    flex-direction: column;
    gap: 8px;
    text-align: center;
  }
  .top-wrapper {
    display: flex;
    gap: 8px;
    justify-content: center;
  }
</style>
