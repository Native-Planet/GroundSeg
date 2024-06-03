<script>
  import { wide } from '$lib/stores/display';
  import { goto } from '$app/navigation';
  import { URBIT_MODE } from '$lib/stores/data'

  $: pfx = $URBIT_MODE ? "/apps/groundseg" : ""

</script>

<div class="card-wrapper {wide ? "wide" : "slim"}">
  <div class="main-title">LOAD A SHIP</div>
  <div class="cards">
    <!-- new ship -->
    <div class="card" on:click={()=>goto("boot/new")} class:disabled={$URBIT_MODE && (type == "existing")}>
      <div class="arrow"></div>
      <div class="title">
        <div class="icon"><img alt="import icon" src={pfx+"/newship-icon.svg"} /></div>
        <div class="title-text">NEW SHIP</div>
      </div>
      <div class="subtitle">
        Load a new moon, planet, star, or galaxy using your bootfile
      </div>
    </div>
    <!-- upload pier -->
    <div class="card" on:click={()=>goto("boot/existing")} class:disabled={$URBIT_MODE}>
      <div class="arrow"></div>
      <div class="title">
        <div class="import icon"><img alt="import icon" src={pfx+"/import-icon.svg"} /></div>
        <div class="title-text">UPLOAD PIER</div>
      </div>
      <div class="subtitle">
        Import an existing pier of a ship
      </div>
    </div>
    <div class="card" on:click={()=>goto("boot/transload")}>
      <div class="arrow"></div>
      <div class="title">
        <div class="transload icon"><img alt="import icon" src={pfx+"/import-icon.svg"} /></div>
        <div class="title-text">ADVANCED IMPORT</div>
      </div>
      <div class="subtitle">
        Transload on-device pier into GroundSeg
      </div>
    </div>
  </div>
</div>

<style>
  .card-wrapper {
    background: var(--bg-base);
    border-radius: 16px;
    margin: auto;
    height: 80vh;
    padding: 40px;
    width: calc(1104px - 80px);
    align-items: center;
    gap: 40px;
  }
  .cards {
    display: flex;
    flex-direction: column;
    gap: 32px;
    align-items: center;
  }
  .main-title {
    text-align: center;
    font-family: var(--title-font);
    color: #000;
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 48px;
    font-weight: 700;
    line-height: 64px; /* 133.333% */
    letter-spacing: -2.88px;
    text-transform: uppercase;
    margin-bottom: 32px;
  }
  .card {
    flex: 1;
    position: relative;
    background-color: var(--bg-modal);
    color: var(--text-color);
    border-radius: 24px;
    transition: background-color, 318ms;
    display: flex;
    flex-direction: column;
    padding: 32px;
    cursor: pointer;
    width: 70%;
  }
  .card:hover {
    background-color: var(--text-card-color);
    cursor: pointer;
  }
  .card:hover .arrow {
    background-image: url('/arrow.svg');
    background-color: var(--text-color);
    cursor: pointer;
  }
  .title {
    display: flex;
    gap: 16px;
    align-items: center;
    margin-bottom: 24px;
  }
  .title-text {
    font-family: var(--title-font);
    leading-trim: both;
    text-edge: cap;
    font-family: BPdotsUnicase;
    font-size: 32px;
    font-style: normal;
    font-weight: 700;
    line-height: 48px; /* 150% */
    letter-spacing: -1.92px;
    text-transform: uppercase;
  }
  .subtitle {
    color: var(--NP_Black, #161D17);
    leading-trim: both;
    text-edge: cap;
    font-family: Inter;
    font-size: 24px;
    font-style: normal;
    font-weight: 300;
    letter-spacing: -1.44px;
    flex: 1;
  }
  .arrow {
    font-family: var(--title-font);
    background-image: url('/arrow-black.svg');
    background-color: var(--bg-boot-arrow);
    background-repeat: no-repeat;
    background-position: center;
    position: absolute;
    bottom: 0;
    right: 0;
    width: 74px;
    height: 56px;
    border-radius: 24px 0;
  }
  .icon {
    margin-top: 12px;
  }
  .transload {
    transform: rotate(270deg);
  }
</style>
