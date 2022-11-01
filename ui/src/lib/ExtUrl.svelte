<script>
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
	export let link
</script>
<a href={link} target="_blank">
  <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
</a>
<style>
a {
  margin: auto;
  margin-left: 16px;
  opacity: .8;
	color: inherit;
}
</style>
