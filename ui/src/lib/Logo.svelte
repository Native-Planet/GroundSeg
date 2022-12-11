<script>
  import { blur } from 'svelte/transition'

  export let t = ""
</script>

<div class="logo" transition:blur={{duration: 400, amount: 100}} >
  <a href="/"><img src="/nplogo.svg" alt="Native Planet Logo" /></a>
	{#if t.length > 0}
		<div class="wrapper"> 
			<div class="text">{t}</div>
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
    padding-bottom: 10px;

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
