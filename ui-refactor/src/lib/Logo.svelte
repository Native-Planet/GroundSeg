<script>
  import { blur } from 'svelte/transition'

  import Fa from 'svelte-fa'
  import { faIgloo } from '@fortawesome/free-solid-svg-icons'

  export let t = ""
</script>

<div class="logo">
  <a href="/"><img transition:blur={{duration: 1000, amount: 100}} src="/nplogo.svg" alt="Native Planet Logo" /></a>
	{#if t.length > 0}
		<div class="wrapper"> 
			<div class="text" transition:blur={{duration: 600, amount: 100}}>{t}</div>
			<a class="home" href="/" transition:blur={{duration: 600, delay: 600, amount: 100}}>
				<Fa icon={faIgloo} />
				home
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
		font-size: 12px;
		margin-left: 18px;
		margin-top: 2px;
		color: inherit;
	}
</style>
