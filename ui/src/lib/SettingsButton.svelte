<script>
  import { page } from '$app/stores'

  const hide = () => {
    return (($page.routeId == 'settings') || ($page.routeId == 'login'))
  }
</script>

<a href='/settings'>
  <img src="/settings.svg" alt="Settings Icon" class:inverted={hide()}/>
</a>

<style>
  .inverted {
    opacity: .2;
    pointer-events: none;
    filter: blur(2px);
  }
	a {
		position:absolute;
		top: 0;
		left: 0;
	}
  img {height: 24px; margin: 20px}
</style>
