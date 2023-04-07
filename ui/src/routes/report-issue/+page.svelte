<script>
  import { onMount } from 'svelte'
	import { updateState, api, urbits } from '$lib/api'

  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import BugPier from '$lib/BugPier.svelte'

	export let data
	updateState(data)

  let description = ''
  let buttonStatus = 'standard'
  let person = ''
  let pierLogs = []
  let bugChecker = []
  let selectAll

  onMount(()=> {
    if (data['status'] == 404) {
      window.location.href = "/login"
    } else {
      getUrbits()
    }
  })

  const forceSet = b => {
    for (let i = 0; i < bugChecker.length; i++) {
      bugChecker[i].forceSet(b)
    }
  }

  const submitReport = () => {
    buttonStatus = 'loading'
    fetch($api + '/bug', {
      method: 'POST',
      credentials: "include",
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'person':person,'message':description.trim(),'logs':pierLogs})
	})
		.then(raw => raw.json())	
    .then(res => { 
      if (res == 200) {
        buttonStatus = 'success'
        setTimeout(()=> person = '', 3000)
        setTimeout(()=> description = '', 3000)
        setTimeout(()=> forceSet(false), 3000)
        setTimeout(()=> selectAll.forceSet(false), 3000)
      } else {
        buttonStatus = 'failure'
      }
      setTimeout(()=> buttonStatus = 'standard', 3000)
    })
    .catch(err => {
      console.log(err)
      buttonStatus = 'failure'
      setTimeout(()=> buttonStatus = 'standard', 3000)
    })
	}

  let succeeded = false
  const getUrbits = () => {
    if (!succeeded) {
      fetch($api + '/urbits', {credentials:"include"})
      .then(raw => raw.json())
      .then(res => {
        updateState(res)
        succeeded = true
      })
      .catch(err => {
        console.log(err)
        if ((typeof err) == 'object') {
          updateState({status:'noconn'})
        }
      })
      setTimeout(getUrbits, 3000)
    }
  }

  const addPier = e => {
    const { name, check } = e.detail
    if (check) {
      if (!pierLogs.includes(name)) {
              pierLogs.push(name);
      }
    } else {
      const index = pierLogs.indexOf(name);
      if (index > -1) {
        pierLogs.splice(index, 1);
      }
    }
  }

  const handleCheckAll = e => {
    forceSet(e.detail.check)
  }

</script>

<Card width="540px">
  <Logo t="Report an issue to Native Planet"/>
  <div class="disclaimer">
    <div class="title">Information sent to Native Planet:</div>
    <ul>
      <li>GroundSeg logs</li>
      <li>StarTram information (if available)</li>
      <li>List of docker containers on your device</li>
      <li>GroundSeg and docker container configs (removed private information)</li>
    </ul>
    {#if $urbits.length > 0}
      <div class="title">Send Pier Logs (Optional):</div>
      <div class="check-wrapper">
        {#each $urbits as u, i}
          <BugPier bind:this={bugChecker[i]} name={u.name} on:update={addPier} submitting={!(buttonStatus == 'standard')}/>
        {/each}
        {#if $urbits.length > 1}
          <BugPier bind:this={selectAll} on:update={handleCheckAll} checkAll={true} submitting={!(buttonStatus == 'standard')}/>
        {/if}
      </div>
    {/if}

  </div>
  <div class="input-title">How should we contact you?</div>
  <input type="text" bind:value={person} placeholder="~sampel-palnet or example@email.com" />
  <div class="input-title">Describe your issue. Include as much information as you can:</div>
  <textarea bind:value={description} placeholder="eg. ~zod doesn't turn on when I try to toggle the switch.."/>
  <div class="submit">
    <PrimaryButton
      on:click={submitReport}
      noMargin={true}
      status={(person.length > 0) && (description.length > 0) ? buttonStatus : "disabled"}
      standard="Submit Report"
      loading="Submitting your bug report..."
      success="Bug report sent!"
      failure="Something went wrong"
    />
  </div>
</Card>

<style>
  .title {
    font-size: 14px; 
  }
  ul {
    font-size: 12px;
    margin: none;
  }
  .disclaimer {
    margin: 20px;
  }
  .input-title {
    font-size: 14px; 
  }
  input {
    margin-top: 12px;
    margin-bottom: 24px;
    width: calc(100% - 40px);
    color: inherit;
    font-family: inherit;
    resize: none;
    background: #ffffff4d;
    border: none;
    border-radius: 8px;
    padding: 8px 20px 8px 20px;
  }
  input:focus {
    outline: none;
  }
  input::placeholder {
    color: inherit;
  }
  textarea {
    margin-top: 12px;
    width: calc(100% - 40px);
    color: inherit;
    font-family: inherit;
    height: 240px;
    resize: none;
    background: #ffffff4d;
    border: none;
    border-radius: 16px;
    padding: 20px;
  }
  textarea:focus {
    outline: none;
  }
  textarea::placeholder {
    color: inherit;
  }
  .submit {
    margin-top: 24px;
    text-align: center;
  }
  .check-wrapper {
    margin-top: 12px;
  }
</style>
