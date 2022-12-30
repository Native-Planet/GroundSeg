<script>
  import { afterUpdate } from 'svelte'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faBug } from '@fortawesome/free-solid-svg-icons'

  let hide = false, blur = false

  afterUpdate(()=> {
    hide = ($page.routeId == 'login')
    blur = ($page.routeId == 'report-issue')
  })

</script>

<a href='/report-issue' class:hide={hide} class:blur={blur}>
  <div class="img">
    <Fa icon={faBug} size="1.2x" />
  </div>
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
    right: 0px;
    top: 3px;
	}
  .img {height: 24px; margin: 20px; color: white}
</style>
