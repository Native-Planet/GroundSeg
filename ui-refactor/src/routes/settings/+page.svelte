<script>
  import { fade } from 'svelte/transition'
  import Logo from '$lib/Logo.svelte'
  import {fadeIn, fadeOut } from '$lib/api'
	import Card from '$lib/Card.svelte'
</script>

<Card width="880px">
  <Logo t='System Settings'/>
</Card>
