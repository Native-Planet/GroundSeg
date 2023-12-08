<script>
  import { setupSkip, setupStarTram } from '$lib/stores/websocket'
  import { structure } from '$lib/stores/data'
  let key = ''
  let activeRegion = "us-east";

  $: regions = ($structure?.regions) || {}
  $: regionKeys = Object.keys(regions)

  const setRegion = r => {
    activeRegion = r
  }
</script>
<div class="title">STARTRAM SERVICE</div>
<div class="wrapper">
  <div class="info">
    <div class="price">$4 / mo</div>
    <div class="promo">Billed annually or $5 month-to-month</div>
    <div class="why">Access your urbit from anywhere - Run multiple urbits from one device - Hassle-free image hosting</div>
  </div>
  {#if regionKeys.length > 0}
    <div class="activate">
      <div class="name">Select Region</div>
      <div class="regions">
        {#each regionKeys as r }
          <div on:click={()=>setRegion(r)} class="region" class:highlight={r == activeRegion}>
            {regions[r].desc}
          </div>
        {/each}
      </div>
    </div>
  {/if}
  <div class="activate">
    <div class="name">Activation Key</div>
    <input placeholder="NativePlanet-some-word-another-word" type="password" bind:value={key}/>
    <button
      disabled={key.length < 1} 
      on:click={()=>setupStarTram(key,activeRegion)}
      >Activate
    </button>
  </div>
  <div class="get">Don't have a key? <a href="https://www.nativeplanet.io/startram" target="_blank">Get one here</a></div>
</div>
<div on:click={setupSkip} class="skip">Skip for now</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    flex: 1;
  }
  .title {
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: var(--title-font);
    font-size: 48px;
    font-style: normal;
    font-weight: 700;
    line-height: normal;
    letter-spacing: -1.92px;
    margin-bottom: 32px;
  }
  .info {
    display: flex;
    flex-direction: column;
    justify-content: end;
    align-items: center;
    width: 420px;
  }
  .price {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 36px;
    font-style: normal;
    font-weight: 500;
    line-height: normal;
    letter-spacing: -1.44px;
    margin-bottom: 8px;
  }
  .promo {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
  }
  .why {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    text-align: center;
  }
  .activate {
    margin-top: 32px;
    width: 100%;
  }
  .name {
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    margin-bottom: 16px;
  }
  input {
    min-width: 600px;
    max-width: 100vw;
    line-height: 59px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    padding-left: 20px;
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  input:active, :focus {
    outline: none; 
  }
  button {
    background: var(--btn-special);
    padding: 0px 48px;
    height: 65px;
    border: none;
    border-radius: 16px;
    color: var(--text-card-color);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  button:hover {
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
  .get {
    margin-top: 32px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    font-weight: 300;
  }
  a {
    color: var(--text-color);
    text-decoration: underline;
  }
  .skip {
    cursor: pointer;
    text-decoration: underline;
    margin-top: 32px;
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 18px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
  }
  .regions {
    display: flex;
    gap: 20px;
  }
  .region {
    flex: 1;
    border-radius: 16px;
    border: solid 2px var(--btn-secondary);
    color: var(--NP_Black, #313933);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    line-height: normal;
    letter-spacing: -1.44px;
    text-align: center;
    height: 61px;
    line-height: 61px;
  }
  .region:hover {
    cursor: pointer;
  }
  .highlight {
    background-color: var(--btn-secondary);
    color: var(--text-card-color);
  }
</style>
