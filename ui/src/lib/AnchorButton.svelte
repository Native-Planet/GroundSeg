<script>
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faSatelliteDish } from '@fortawesome/free-solid-svg-icons'

  const hide = () => {
    return (($page.routeId == 'anchor') || ($page.routeId == 'login'))
  }

</script>

<a href='/anchor'>
  <div class="img" class:inverted={hide}>
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
    filter: blur(2px);
  }
	a {
		position:absolute;
		left: 44px;
    top: 3px;
	}
  .img {height: 24px; margin: 20px; color: white;}
</style>
