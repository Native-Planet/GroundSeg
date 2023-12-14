<script>
  import './profile.css'
  import { wide } from '$lib/stores/display'
  import Password from './password.svelte'
  import StarTram from './startram.svelte'
</script>

<style>
  .spacer {height: 40px;}
</style>

<Password {wide} />
<div class="spacer"></div>
<StarTram />
<div class="spacer"></div>
