<script>
  import './profile.css'
  import { wide } from '$lib/stores/display'
  import Password from './Password.svelte'
  import StarTram from './StarTram.svelte'
  import Hermes from './Hermes.svelte'
</script>

<style>
  .spacer {height: 40px;}
</style>

<Password {wide} />
<div class="spacer"></div>
<StarTram />
<div class="spacer"></div>
<Hermes />
<div class="spacer"></div>
