<script>
  import { api } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
	import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import Clipboard from 'clipboard'

  export let endpoint = '', accessKey = ''

	const dispatch = createEventDispatcher();

  const handleClick = () =>	dispatch('click')

  let secret = ''
  let bucket = 'bucket'
  let copyCMD1, copyCMD2, copyCMD3, copyCMD4
  let clickedCMD1 = false
  let clickedCMD2 = false
  let clickedCMD3 = false
  let clickedCMD4 = false
  let shown = true

  let ep = endpoint.substring(endpoint.indexOf('.')+1)

  let dojoCommand1 = ':s3-store|set-endpoint \'' + ep + '\'\n'
  let dojoCommand2 = ':s3-store|set-access-key-id \'' + accessKey + '\'\n'
  let dojoCommand3 = ':s3-store|set-secret-access-key \'' + secret + '\'\n'
  let dojoCommand4 = ':s3-store|set-current-bucket \'' + bucket + '\'\n'

  onMount(()=> {
    shown = true
    getMinIOSecret()
    copyCMD1 = new Clipboard('#one')
    copyCMD2 = new Clipboard('#two')
    copyCMD3 = new Clipboard('#three')
    copyCMD4 = new Clipboard('#four')

    copyCMD1.on("success", ()=> {
    clickedCMD1 = true; setTimeout(()=> clickedCMD1 = false, 3000)})
    copyCMD2.on("success", ()=> {
    clickedCMD2 = true; setTimeout(()=> clickedCMD2 = false, 3000)})
    copyCMD3.on("success", ()=> {
    clickedCMD3 = true; setTimeout(()=> clickedCMD3 = false, 3000)})
    copyCMD4.on("success", ()=> {
    clickedCMD4 = true; setTimeout(()=> clickedCMD4 = false, 3000)})

  })

  onDestroy(()=> shown = false)

  const getMinIOSecret = () => {
    if (shown) {
    let u = $api + "/urbit/minio_secret"
    const f = new FormData()
    f.append('patp',accessKey)
      fetch(u, {method: 'POST',body: f}).then(r => r.json()).then(d => {
        secret = d
        dojoCommand1 = ':s3-store|set-endpoint \'' + ep + '\'\n'
        dojoCommand2 = ':s3-store|set-access-key-id \'' + accessKey + '\'\n'
        dojoCommand3 = ':s3-store|set-secret-access-key \'' + secret + '\'\n'
        dojoCommand4 = ':s3-store|set-current-bucket \'' + bucket + '\'\n'

        setTimeout(getMinIOSecret, 1000) })}}

</script>

<div class="container">
  <div class="title">Update MinIO endpoint</div>
  <div class="header">
    We are working on getting this procedure automated!
  </div>
  <div class="instruction">
    <div class="num">1.</div>
    <div class="words">Copy the commands given below.</div>
  </div>

  <div class="instruction">
    <div class="num">2.</div>
    <div class="words">Enter your Urbit and navigate to Terminal.</div>
  </div>
  <div class="instruction">
    <div class="num">3.</div>
    <div class="words">Right click and select 'paste'</div>
  </div>
  <div class="instruction">
    <div class="num">4.</div>
    <div class="words">Hit the 'enter' key.</div>
  </div>


  <div class="dojo-commands" data-clipboard-text={dojoCommand1} id="one">
    {clickedCMD1 ? 'copied!' : dojoCommand1}
  </div>
  <div class="dojo-commands" data-clipboard-text={dojoCommand2} id="two">
    {clickedCMD2 ? 'copied!' : dojoCommand2}
  </div>
  <div class="dojo-commands" data-clipboard-text={dojoCommand3} id="three">
    {clickedCMD3 ? 'copied!' : dojoCommand3}
  </div>
  <div class="dojo-commands" data-clipboard-text={dojoCommand4} id="four">
    {clickedCMD4 ? 'copied!' : dojoCommand4}
  </div>

  <PrimaryButton
    standard="Back to profile"
    status="standard"
    left={false}
    on:click={handleClick} />


</div>

<style>

  .container {
    margin: 40px;
  }
  .title {
    font-size: 18px;
    margin-bottom: 24px;
  }
  .header {
    font-size: 14px;
    margin-bottom: 12px;
    font-style: italic;
  }
  .instruction {
    display: flex;
    font-size: 14px;
    opacity: .6;
  }
  .num {
    width: 16px;
  }
  .words {
    flex: 1;
  }
  .dojo-commands {
    white-space: pre-wrap;
    width: 460px;
    line-height: 16px;
    font-size: 12px; 
    margin: 12px 0 12px 0;
    background: #ffffff4d;
    padding: 8px 12px 8px 12px;
    cursor: pointer;
    border-radius: 6px;

  }

</style>
