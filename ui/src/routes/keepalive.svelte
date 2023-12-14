<script>
  import { onMount } from 'svelte'
  import { alive, lastActivity } from './alive';
  import { sendHeartbeat } from '$lib/stores/urbit'

  $: handleActivity($alive)
  let sendOnActivity = false

  const handleActivity = (n) => {
    lastActivity.set(new Date())
    if (sendOnActivity) {
      sendOnActivity = false
      sendHeartbeat()
    }
  }

  const handleHeartbeat = () => {
    let date = new Date()
    if (date - $lastActivity < 30000) {
      sendHeartbeat()
    } else {
      sendOnActivity = true
    }
    setTimeout(handleHeartbeat, 10000)
  }


  onMount(()=>{
    handleHeartbeat()
  })
</script>
<div></div>

<style>
  div {
    pointer-events: none;
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
  }
</style>
