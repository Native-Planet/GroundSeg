<script>
  import { afterUpdate } from 'svelte'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faIgloo } from '@fortawesome/free-solid-svg-icons'

  let hide = false

  afterUpdate(()=> {
    hide = ($page.route.id == '/login')
  })

</script>

<a href='/' class:hide={hide}>
  <div class="img">
    <Fa icon={faIgloo} size="1.2x" />
  </div>
</a>

<style>
  .hide {
    opacity: 0;
    pointer-events: none;
  }
	a {
		position:absolute;
		left: 88px;
    top: 3px;
	}
  .img {height: 24px; margin: 20px; color: white}
</style>
