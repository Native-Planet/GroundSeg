<script>
  import { onMount } from 'svelte'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
	import { updateState, api } from '$lib/api'

	export let data
	updateState(data)

  let description = '', buttonStatus = 'standard', person = ''

  onMount(()=> {
    if (data['status'] == 404) {
      window.location.href = "/login"
    }
  })

  const submitReport = () => {
    buttonStatus = 'loading'
    fetch($api + '/bug', {
      method: 'POST',
      credentials: "include",
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({'person':person,'message':description.trim()})
	})
		.then(raw => raw.json())	
    .then(res => { 
      if (res == 200) {
        buttonStatus = 'success'
        setTimeout(()=> person = '', 3000)
        setTimeout(()=> description = '', 3000)
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


</script>

<Card width="540px">
  <Logo t="Report an issue to Native Planet"/>
  <div class="disclaimer">
    <div class="title">Information sent to Native Planet:</div>
    <ul>
      <li>GroundSeg logs</li>
      <li>StarTram information (if available)</li>
      <li>List of docker containers on your device</li>
      <li>GroundSeg and docker container configurations (sensitive information not sent)</li>
    </ul>
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
</style>
