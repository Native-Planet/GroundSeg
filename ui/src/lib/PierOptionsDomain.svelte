<script>
  import { api } from '$lib/api'

  import Fa from 'svelte-fa'
  import { faCircleQuestion } from '@fortawesome/free-regular-svg-icons'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

  export let name, alias, title, svcType, stdText

  let removeStatus = "standard"
  let saveStatus = "standard"
  let customDomain = alias
  let info = false
  let relinkCheck = true
    
  const submit = () => {
    saveStatus = 'loading'
		fetch($api + '/urbit?urbit_id=' + name, {
    method: 'POST',
    credentials: "include",
    headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({
        'app':'cname',
        'data': {
          'svc_type': svcType,
          'alias':customDomain,
          'operation': 'create',
          'relink': relinkCheck
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
          'svc_type': svcType,
          'alias':customDomain,
          'operation': 'delete',
          'relink': relinkCheck
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

<div class="bg">
  <div class="option-title">{title}
    <button class="question-mark" on:click={()=> info = !info} >
      <Fa icon={faCircleQuestion} size="1.2x" />
    </button>
    {#if info}
      <slot />
    {/if}
  </div>
  <div class="panel">
    <input type="text" class="domain" spellcheck="false" bind:value={customDomain} placeholder="custom.domain.com"/>
    {#if svcType == 'minio'}
      <div class="relink-check" on:click={()=> relinkCheck = !relinkCheck}>
        <div class="box" class:highlight={relinkCheck}>
          {#if relinkCheck}
            <Fa icon={faCheck} size="1x"/>
          {/if}
        </div>
        Automatically link to Urbit
      </div>
    {/if}
    <div class="button-wrapper">
      <PrimaryButton
        noMargin={true}
        background="#FFFFFF4D"
        standard="Remove"
        loading="removing"
        success="custom domain removed!"
        failure="failed to remove custom domain"
        status={ (alias == customDomain) && (alias.length > 0) ? removeStatus : 'disabled'}
        on:click={remove}
        />
      <PrimaryButton
        noMargin={true}
        standard={stdText}
        loading="Registering"
        success="Custom domain registered!"
        failure="failed to register domain"
        status={alias != customDomain ? saveStatus : 'disabled'}
        on:click={submit}
      />
    </div>
  </div>
</div>

<style>
  .bg {
    background: #0000001d;
    padding: 20px 0 20px 0;
    border-radius: 12px;
  }
  .option-title {
    width: 100%;
    text-align: center;
    font-size: 14px;
    padding-bottom: 12px;
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
  .relink-check {
    display: flex;
    gap: 6px;
    align-items: center;
    justify-content: center;
    text-align: center;
    font-size: 11px;
    padding-bottom: 6px;
    cursor: pointer;
    user-select: none;
  }
  .box {
    width: 14px;
    height: 14px;
    background: #ffffff4d;
    border-radius: 4px;
  }
  .highlight {
    background: #028AFB;
  }
</style>
