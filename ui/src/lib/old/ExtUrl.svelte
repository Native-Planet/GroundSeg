<script>
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'
	export let link
  export let advanced = false
</script>
<a style="{advanced ? "margin-top:12px" : "margin:auto;margin-left:16px"};" href={link} target="_blank">
  <Fa icon={faArrowUpRightFromSquare} size="{advanced ? "1x" : "1.2x"}" />
</a>
<style>
a {
  opacity: .8;
	color: inherit;
}
</style>
