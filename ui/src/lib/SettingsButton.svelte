<script>
  import { page } from '$app/stores'
</script>

<a href='/settings'>
  <img src="/settings.svg" alt="Settings Icon" class:inverted={$page.routeId == 'settings'}/>
</a>

<style>
  .inverted {
    filter: invert(1);
    pointer-events: none;
  }
	a {
		position:absolute;
		top: 0;
		left: 0;
	}
  img {height: 24px; margin: 20px}
</style>
