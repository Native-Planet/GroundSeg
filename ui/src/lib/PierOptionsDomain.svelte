<script>
  import { api } from '$lib/api'

  import Fa from 'svelte-fa'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, alias

  let removeStatus = "standard"
  let saveStatus = "standard"
  let customDomain = alias
  let info = false
    
  const submit = () => {
    saveStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
    method: 'POST',
    credentials: "include",
    headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        'app':'cname',
        'data': {
          'svc_type':'urbit-web',
          'alias':customDomain,
          'operation': 'create'
        }
      })
    })
      .then(d=>d.json())
      .then(r=> {
        console.log(r)
        if (r == 200) {
          saveStatus = 'success'
          setTimeout(()=>{
            saveStatus = 'standard'
            customDomain = alias
          }, 3000)
        } else {
          saveStatus = 'failure'
          console.log(r)
          setTimeout(()=>{
            saveStatus = 'standard'
            customDomain = alias
          }, 3000)
        }
      })
      .catch(err => {
        console.log(err)
        saveStatus = 'failure'
        setTimeout(()=>{
          saveStatus = 'standard'
          customDomain = alias
        }, 3000)
      })
  }

  const remove = () => {
    removeStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
    method: 'POST',
    credentials: "include",
    headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        'app':'cname',
        'data': {
          'svc_type':'urbit-web',
          'alias':customDomain,
          'operation': 'delete'
        }
      })
    })
      .then(d=>d.json())
      .then(r=> {
        console.log(r)
        if (r == 200) {
          removeStatus = 'success'
          customDomain = ''
          setTimeout(()=>{
            removeStatus = 'standard'
          }, 3000)
        } else {
          removeStatus = 'failure'
          console.log(r)
          setTimeout(()=>{
            removeStatus = 'standard'
          }, 3000)
        }
      })
      .catch(err => {
        console.log(err)
        removeStatus = 'failure'
        setTimeout(()=>{
          removeStatus = 'standard'
        }, 3000)
      })
  }
</script>

<div class="option-title">Custom Domain
  <button class="question-mark" on:click={()=> info = !info} >
    <Fa icon={faCircleQuestion} size="1.2x" />
  </button>
  {#if info}
    <div class="info">
      This allows you to access your Urbit ship from a second domain. Please read
      <a href="https://www.nativeplanet.io/custom-startram-domains" target="_blank">this guide</a>
      for more information.
    </div>
  {/if}
</div>
<div class="panel">
  <input type="text" class="domain" spellcheck="false" bind:value={customDomain} placeholder="custom.domain.com"/>
  <div class="button-wrapper">
    <PrimaryButton
      noMargin={true}
      background="#BD4140"
      standard="Remove"
      loading="removing"
      success="custom domain removed!"
      failure="failed to remove custom domain"
      status={ alias == customDomain ? removeStatus : 'disabled'}
      on:click={remove}
      />
    <PrimaryButton
      noMargin={true}
      standard="Submit domain"
      loading="Registering"
      success="Custom domain registered!"
      failure="failed to register domain"
      status={alias != customDomain ? saveStatus : 'disabled'}
      on:click={submit}
    />
  </div>
</div>

<style>
  .option-title {
    width: 100%;
    text-align: center;
    font-size: 14px;
    color: inherit;
  }
  .domain {
    font-size: 12px;
    font-family: inherit;
    color: inherit;
    width: 80%;
    background: #ffffff4d;
    outline: none;
    border: none;
    padding: 6px;
    border-radius: 6px;
    margin-bottom: 8px;
    text-align: center;
  }
  .domain::placeholder {
    color: inherit;
    opacity: .6;
  }
  .button-wrapper {
    display: flex;
    gap: 12px;
    justify-content: center;
    align-items: end;
    padding: 4px;
  }
  .question-mark {
    color: inherit;
    cursor: pointer;
  }
  .info {
    font-size: 11px;
  }
  a {
    color: inherit;
    text-decoration: underline;
  }

</style>
