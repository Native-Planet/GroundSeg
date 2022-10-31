<script>
  import { blur } from 'svelte/transition'

  import Fa from 'svelte-fa'
  import { faIgloo } from '@fortawesome/free-solid-svg-icons'

  export let t = ""
</script>

<div class="logo" transition:blur={{duration: 400, amount: 100}} >
  <a href="/"><img src="/nplogo.svg" alt="Native Planet Logo" /></a>
	{#if t.length > 0}
		<div class="wrapper"> 
			<div class="text">{t}</div>
			<a class="home" href="/">
				<Fa icon={faIgloo} />
        <span>home</span>
			</a>
		</div>
	{/if}
</div>

<style>
  .logo {
    display: flex;
    align-items: end;
  }
  img {
    height: 32px;
  }
	.wrapper {
	}
  .text {
    font-size: 14px;
    padding-left: 18px;

	}
	.home {
    display: flex;
		font-size: 12px;
		margin-left: 18px;
		margin-top: 5px;
		color: inherit;
    gap: 4px;
	}
</style>
