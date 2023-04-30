<script>
  import { api } from '$lib/api'
  import { send, socket, socketInfo } from '$lib/stores/websocket.js'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  import PierAdvancedMinIOSetup from '$lib/PierAdvancedMinIOSetup.svelte'
  import PierAdvancedMinIO from '$lib/PierAdvancedMinIO.svelte'

  export let minIOReg
  export let minIOUrl
  export let hasBucket
  export let name
  export let disabled = false

  let showSetup = false

  $: linkInfo = $socketInfo.urbits[name].minio.link
  $: unlinkInfo = $socketInfo.urbits[name].minio.unlink

  // Button status
  let linkButtonStatus = 'standard'
  let unlinkButtonStatus = 'standard'
  let exportBucketStatus = 'standard'

	// Update Urbit S3 endpoint
	const updateMinIO = () => {
    let payload = {
      "category": "urbits",
      "payload": {"patp": name, "module": "minio", "action": "link"}
    }
    send($socket, $socketInfo, document.cookie, payload)
  }

	const unlinkMinIO = () => {
    let payload = {
      "category": "urbits",
      "payload": {"patp": name, "module": "minio", "action": "unlink"}
    }
    send($socket, $socketInfo, document.cookie, payload)
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
  {#if showSetup}
    <PierAdvancedMinIOSetup {name} {minIOReg} on:cancel={()=>showSetup = false} />
  {:else}
    <div class="wrapper">
      {#if minIOReg}
        <PierAdvancedMinIO {minIOUrl} />
      {:else}
        <PrimaryButton
          noMargin={true}
          standard="Setup MinIO Local Storage"
          on:click={()=> showSetup = true}
        />
      {/if}
      <div class="mid-wrapper">
        {#if linkInfo.length <= 0}
          <PrimaryButton
            noMargin={true}
            standard="Link to Urbit"
            status={disabled || !minIOReg ? "disabled" : linkButtonStatus}
            on:click={updateMinIO} />
        {:else}
          {#if linkInfo == "create-account"}
            <div class="link-info">creating secret key</div>
          {/if}
          {#if linkInfo == "link-click"}
            <div class="link-info">trying with click</div>
          {/if}
          {#if linkInfo == "link-lens"}
            <div class="link-info orange">trying with lens</div>
          {/if}
          {#if linkInfo == "success"}
            <div class="link-info lime">linked!</div>
          {/if}
          {#if linkInfo == "failure"}
            <div class="link-info red">failed to link</div>
          {/if}
        {/if}
        {#if unlinkInfo.length <= 0}
          <PrimaryButton
            noMargin={true}
            background="#FFFFFF4D"
            standard="Unlink"
            status={disabled || !minIOReg ? "disabled" : unlinkButtonStatus}
            on:click={unlinkMinIO}
          />
        {:else}
          {#if unlinkInfo == "link-click"}
            <div class="link-info">trying with click</div>
          {/if}
          {#if unlinkInfo == "link-lens"}
            <div class="link-info orange">trying with lens</div>
          {/if}
          {#if unlinkInfo == "success"}
            <div class="link-info lime">unlinked!</div>
          {/if}
          {#if unlinkInfo == "failure red"}
            <div class="link-info">failed to unlink</div>
          {/if}
        {/if}
      </div>
      <PrimaryButton
        noMargin={true}
        background="#FFFFFF4D"
        standard="Export Bucket"
        loading="Compressing your files.."
        status={hasBucket && minIOReg ? exportBucketStatus : "disabled"}
        on:click={exportBucket}
      />
    </div>
  {/if}
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
  .mid-wrapper {
    display: flex;
    gap: 8px;
    justify-content: center;
  }
  .link-info {
    font-size: 12px;
    margin: auto 0;
    width: 95px;
    animation: breathe 2s infinite;
  }
  .red {
    color: red;
  }
  .orange {
    color: orange;
  }
  .lime {
    color: lime;
    animation: none;
  }
</style>
