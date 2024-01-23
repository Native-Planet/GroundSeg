<script>
  import { structure, daysUntilDate } from '$lib/stores/data'

  import Modal from '$lib/Modal.svelte'
  import { closeModal } from 'svelte-modals'

  export let isOpen

  $: startram = $structure?.profile?.startram || {}
  $: urlID = startram?.info?.urlID || ""
  $: endpoint = startram?.info?.endpoint || ""

</script>

<Modal width={720}>
  {#if isOpen}
  <div class="wrapper">
    <div class="bill">
      <div class="price">$50/year</div>
      <div class="name">Billed annually</div>
      <a href="https://{endpoint}/v1/stripe/convert?url_id={urlID}&term=year" target="_blank">Subscribe</a>
    </div>
    <div class="bill">
      <div class="price">$5/month</div>
      <div class="name">Billed monthly</div>
      <a href="https://{endpoint}/v1/stripe/convert?url_id={urlID}&term=month" target="_blank">Subscribe</a>
    </div>
  </div>
  {/if}
</Modal>

<style>
  .wrapper {
    padding: 32px;
    display: flex;
    gap: 32px;
    align-items: center;
  }
  .bill {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    color: var(--text-color, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    gap: 32px;
    border: solid 2px black;
    border-radius: 16px;
    padding: 32px;
  }
  .price {
    font-size: 36px;
  }
  .name {
    font-size: 18px;
    letter-spacing: 0;
    font-weight: 600;
  }
  a {
    display: inline-flex;
    padding: 24px 48px;
    justify-content: center;
    align-items: center;
    gap: 8px;
    background: #000;
    border-radius: 16px;
    color: #FFF;
    text-align: center;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: 32px; /* 133.333% */
    letter-spacing: -1.44px;
    cursor: pointer;
  }
  a:disabled {
    pointer-events: none;
    opacity: .6;
  }
</style>
