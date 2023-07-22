<script>
  import { wide } from '$lib/stores/display'
  import './profile.css'
  import Password from './Password.svelte'
  import StarTram from './StarTram.svelte'
  import Support from './Support.svelte'
</script>

<Password {wide} />
<div class="spacer"></div>
<StarTram {wide} />
<div class="spacer"></div>
<Support {wide} />
<div class="spacer"></div>
<style>
  .spacer {
    height: 40px;
  }
</style>
