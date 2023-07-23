<script>
  import './profile.css'

  import { wide } from '$lib/stores/display'
  import { showRegisterModal } from './store'

  import Password from './Password.svelte'
  import StarTram from './StarTram.svelte'
  import RegisterModal from './RegisterModal.svelte'
</script>

<StarTram {wide} />
<div class="spacer"></div>
<Password {wide} />
<div class="spacer"></div>
{#if $showRegisterModal}
  <RegisterModal />
{/if}
<style>
  .spacer {
    height: 40px;
  }
</style>
