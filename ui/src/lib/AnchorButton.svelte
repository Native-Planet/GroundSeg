<script>
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faSatelliteDish } from '@fortawesome/free-solid-svg-icons'

</script>

<a href='/anchor'>
  <div class="img" class:inverted={$page.routeId == 'anchor'}>
    <Fa icon={faSatelliteDish} size="1.2x" />
  </div>
  <!--
  <img src="/settings.svg" alt="Settings Icon" />
  -->
</a>

<style>
  .inverted {
    opacity: 0.2;
    pointer-events: none;
  }
	a {
		position:absolute;
		left: 44px;
    top: 3px;
	}
  .img {height: 24px; margin: 20px; color: white;}
</style>
