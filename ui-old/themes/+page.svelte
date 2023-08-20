<script>
	import { onMount, onDestroy } from 'svelte'

  import Logo from '$lib/Logo.svelte'
	import Card from '$lib/Card.svelte'

  let inViewThemes = false

  onMount(()=> {
    inViewThemes = true
  })

	// end the update loop
  onDestroy(()=> inViewThemes = false)

</script>

{#if inViewThemes}
  <Card width="800px">
    <Logo t='GroundSeg Theme'/>
  </Card>
{/if}
