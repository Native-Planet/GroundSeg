<script>
	import { onMount, onDestroy } from 'svelte'
  import { scale } from 'svelte/transition'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faCheck } from '@fortawesome/free-solid-svg-icons'

	import { updateState, api, system, noconn, updateLinux } from '$lib/api'
  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'

	let inView = false
  let buttonStatus = 'standard'

	onMount(()=> inView = true)
  onDestroy(()=> inView = false)

  const updateDevice = () => {
    buttonStatus = 'loading'
	  fetch($api + '/linux/updates', {
			method: 'POST',
      credentials: "include",
	  })
      .then(d=>d.json()).then(r=>{
        if (r === 200) {
          buttonStatus = 'success'
          setTimeout(()=>{
            buttonStatus = 'standard'
          }, 3000)}
        if (r === 400) {
          buttonStatus = 'failure'
          setTimeout(()=>buttonStatus = 'standard', 3000)
   }})}
	
</script>

{#if inView}
  <Card width="460px">
    <Logo t='Device Update'/>
    <div class="linux-wrapper">
      <div class="text">
        Update the underlying operating system of your Native Planet Device.
      </div>
      <PrimaryButton 
        on:click={updateDevice}
        standard={$updateLinux ? "Update and Restart Device" : "No Updates, Run Anyways"}
        status={buttonStatus}
        loading="Updating..."
        failure="Something went wrong"
        />
    </div>
  </Card>
{/if}

<style>
  .linux-wrapper {
    text-align: center;
  }
  .text {
    padding: 40px;
    font-size: 12px;
  }
</style>
