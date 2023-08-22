<script>
  import './profile.css'

  import { wide } from '$lib/stores/display'
  import { showRegisterModal, showEndpointModal, showCancelModal } from './store'

  import Password from './Password.svelte'
  import StarTram from './StarTram.svelte'
  import RegisterModal from './RegisterModal.svelte'
  import EndpointModal from './EndpointModal.svelte'
  import CancelModal from './CancelModal.svelte'
</script>

<Password {wide} />
<div class="spacer"></div>
<StarTram />
<div class="spacer"></div>

{#if $showRegisterModal}
  <RegisterModal />
{/if}
{#if $showEndpointModal}
  <EndpointModal />
{/if}
{#if $showCancelModal}
  <CancelModal />
{/if}
<style>
  .spacer {
    height: 40px;
  }
</style>
