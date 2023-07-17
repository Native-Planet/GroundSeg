<script>
	import { onMount } from 'svelte'
  import { scale } from 'svelte/transition'
	import { page } from '$app/stores'
	import { api } from '$lib/api'
	import Sigil from '$lib/Sigil.svelte'
  import Fa from 'svelte-fa'
  import { faArrowUpRightFromSquare } from '@fortawesome/free-solid-svg-icons'

  export let u
  export let name
	let inView = false
  $: containerStatus = (u?.container?.status) || "loading"
  $: containerUrl = (u?.container?.url) || ""
	onMount(()=> inView = true)

</script>

{#if inView}
  <div class="pier" in:scale={{duration:120, delay: 300}}>
    <Sigil patp={name} size="60px" rad="8px" />
    <a class="info" href={name}>
      <div class="patp">{name}</div>
      <div class="status">{containerStatus.charAt(0).toUpperCase() + containerStatus.slice(1)}</div>
    </a>
    <a class="ext" href={containerStatus == "running" ? containerUrl : ""} target={containerStatus == "running" ? "_blank" : ""}>
      <Fa icon={faArrowUpRightFromSquare} size="1.2x" />
    </a>
  </div>
{/if}

<style>
	a { color: inherit; }
  .ext:hover {
    opacity: .6;
  }
  .pier {
    padding: 6px 20px 6px 20px;
    display: flex;
    align-items: center;
    flex-wrap: wrap;
  }
  .pier:hover {
    background: #00000080;
  }
  .info {
    display: flex;
    flex-direction: column;
    padding-left: 12px;
    flex: auto;
  }
  .patp {
    font-weight: 600;
    color: inherit;
  }
  .status {
    opacity: .8;
    font-weight: 400;
    font-size: .8em;
  }
</style>
