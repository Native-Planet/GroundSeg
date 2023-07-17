<script>
  import { afterUpdate } from 'svelte'
  import { page } from '$app/stores'

  let hide = false
  let blur = false
  afterUpdate(()=> {
    hide = ($page.route.id == '/login')
    blur = ($page.route.id == '/settings')
  })

</script>

<a href='/settings' class:hide={hide} class:blur={blur}>
  <img src="/settings.svg" alt="Settings Icon"/>
</a>

<style>
  .blur {
    opacity: .3;
    pointer-events: none;
  }
  .hide {
    opacity: 0;
    pointer-events: none;
  }
	a {
		position:absolute;
		top: 0;
		left: 0;
	}
  img {height: 24px; margin: 20px}
</style>
