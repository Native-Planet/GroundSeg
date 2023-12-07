<script>
  import { structure, connected, URBIT_MODE } from '$lib/stores/data'
  import { isC2CMode } from '$lib/stores/websocket'
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte'
  import { page } from '$app/stores'

  $: authLevel = ($structure?.auth_level) || "unauthorized"
  $: stage = ($structure?.stage) || null
  $: pageRouteID = $page.route.id

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

  let count = 0

  const redirector = () => {
    if ($connected) {
      if ($URBIT_MODE) {
        urbitRedirect() 
      } else {
        wsRedirect()
      }
    }
    setTimeout(redirector,500)
  }

  const urbitRedirect = () => {
    if (authLevel != "authorized") {
      if (pageRouteID != "/[patp]") {
        goto(pfx + "/" + authLevel)
      }
    } //else {
      // do something
    //}
  }

  const wsRedirect = () => {
    if ($isC2CMode) {
      if (pageRouteID !== (pfx + "/captive")) {
        goto(pfx+"/captive")
      }
    } else {
      const auth = authLevel === "authorized"
      if (auth) {
        wsAuthRedirect()
      } else {
        if (authLevel === "unauthorized") {
          wsUnauthRedirect()
        }
        if (authLevel === "setup") {
          wsSetupRedirect()
        }
      }
    }
  }

  const wsAuthRedirect = () => {
    if ((pageRouteID === (pfx+"/login")) || ($page.route.id.includes("setup"))) {
      goto(pfx+"/")
    }
  }

  const wsUnauthRedirect = () => {
    if (pageRouteID !== (pfx+"/login")) {
      if (count > 2) {
        count = 0
        goto(pfx+"/login")
      } else {
        count += 1 
      }
    }
  }

  const wsSetupRedirect = () => {
    if (count > 2) {
      count = 0
      const currentStage = "/setup/" + stage
      if (currentStage != $page.route.id) {
        goto("/setup/" + stage)
      }
    } else {
      count += 1 
    }
  }

  onMount(()=>{
    redirector()
  })

</script>
