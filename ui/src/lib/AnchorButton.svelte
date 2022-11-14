<script>
  import { afterUpdate } from 'svelte'
  import { page } from '$app/stores'
  import Fa from 'svelte-fa'
  import { faSatelliteDish } from '@fortawesome/free-solid-svg-icons'

  let hide = false
  let blur = false
  afterUpdate(()=> {
    hide = ($page.routeId == 'login')
    blur = ($page.routeId == 'anchor')
  })

</script>

<a href='/anchor' class:hide={hide} class:blur={blur}>
  <div class="img">
    <Fa icon={faSatelliteDish} size="1.2x" />
  </div>
</a>

<style>
  .blur {
    opacity: .2;
    pointer-events: none;
    filter: blur(2px);
  }
  .hide {
    opacity: 0;
    pointer-events: none;
  }
	a {
		position:absolute;
		left: 44px;
    top: 3px;
	}
  .img {height: 24px; margin: 20px; color: white}
</style>
