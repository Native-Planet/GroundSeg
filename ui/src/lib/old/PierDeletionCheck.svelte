<script>
  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

  import { createEventDispatcher } from 'svelte'
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, hasBucket

  let exportBucketStatus = 'standard',
    exportPierStatus = 'standard', 
    deleteButtonStatus = 'standard',
    finalCheck = false

  const dispatch = createEventDispatcher()

  const exportBucket = () => {
    exportBucketStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
      credentials: "include",
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

  const exportUrbitPier = () => {
    exportPierStatus = 'loading'

		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
      credentials: "include",
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'export'})
		  })
      .then(res => {return res.blob()})
      .then(d => {
        exportPierStatus = 'success'
        var a = document.createElement("a")
        a.href = window.URL.createObjectURL(d)
        a.download = name
        a.click()
        setTimeout(()=> exportPierStatus = 'standard', 5000)
  })}

  const deleteData = () => {
    deleteButtonStatus = 'loading'

		fetch($api + '/urbit?urbit_id=' + name, {
  		method: 'POST',
      credentials: "include",
	  	headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'app':'pier','data':'delete'})
		  })
      .then(res => res.json())
      .then(d => { if (d == 200) {
        window.location.href = '/'
      }
  })}

</script>

<div class="redbg">
  <div class="warning">WARNING</div>
  <div class="line">
    Delete your Urbit Pier and all services related to this Urbit ID?
  </div>
  <div class="line">
    This action cannot be undone.
    Please export data you want to save:
  </div>
 
  <div class="export">
    <PrimaryButton
      noMargin={true}
      background="#FFFFFF4D" 
      standard="Export Urbit Pier"
      loading="Compressing your pier..."
      success="Your pier has been exported"
      status={exportPierStatus}
      on:click={exportUrbitPier}
      />

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

  <div class="final-check" on:click={()=> finalCheck = !finalCheck}>
    <div class="box" class:highlight={finalCheck}>
      {#if finalCheck}
        <Fa icon={faCheck} size="1x"/>
      {/if}
    </div>
    I understand that this action cannot be undone.
  </div>
</div>

<div class="buttons">
  <div class="cancel" on:click={()=> dispatch('cancel')}>Cancel</div>
  <PrimaryButton
    left={false}
    background="red" 
    standard="Delete all data related to ~{name}"
    loading="Deleting..."
    status={finalCheck ? deleteButtonStatus : 'disabled'}
    on:click={deleteData}
    />
</div>

<style>
  .redbg {
    background: #3F00008D;
    border-radius: 16px;
    padding: 20px;
    margin-bottom: 20px;
  }
  .warning {
    font-size: 16px;
    padding-bottom: 18px;
    text-align: center;
  }
  .line {
    font-size: 13px;
    padding: 0 40px 18px 40px;
    text-align: center;
  }
  .export {
    display: flex;
    gap: 24px;
    justify-content: center;
    padding-bottom: 18px;
  }
  .final-check {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: center;
    text-align: center;
    font-size: 14px;
    padding: 20px;
    cursor: pointer;
    user-select: none;
  }
  .buttons {
    display: flex;
    align-items: center;
  }
  .box {
    width: 20px;
    height: 20px;
    line-height: 20px;
    background: #ffffff4d;
    border-radius: 4px;
  }
  .highlight {
    background: #028AFB;
  }
  .cancel {
    padding-left: 20px;
    font-size: 12px;
    width: 80px;
    cursor: pointer;
  }
</style>
