<script>
  import { wide } from '$lib/stores/display'
  import './profile.css'
  import Password from './Password.svelte'
  import StarTram from './StarTram.svelte'
</script>

<StarTram {wide} />
<div class="spacer"></div>
<Password {wide} />
<div class="spacer"></div>
<style>
  .spacer {
    height: 40px;
  }
</style>
