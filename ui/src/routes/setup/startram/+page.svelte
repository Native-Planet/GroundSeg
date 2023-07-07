<script>
</script>
<div class="title">STARTRAM SERVICE</div>
<div class="wrapper">
  <div class="info">
    <div class="price">$4 / mo</div>
    <div class="promo">Billed annually or $5 month-to-month</div>
    <div class="why">Access your urbit from anywhere / Run multiple urbits from one device / Hassle-free image hosting</div>
  </div>
  <div class="activate">
    <div class="name">Activation Key</div>
    <input placeholder="NativePlanet-some-word-another-word" type="password" />
    <button>Activate</button>
  </div>
  <div class="get">Don't have a key? <a href="https://www.nativeplanet.io/startram" target="_blank">Get one here</a></div>
</div>
<div class="skip">Skip for now</div>

<style>
  .wrapper {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    flex: 1;
  }
  .title {
    font-size: 48px;
    font-family: var(--title-font);
    line-height: calc(48px * 2);
  }
  .info {
    display: flex;
    flex-direction: column;
    justify-content: end;
    align-items: center;
    width: 420px;
  }
  .price {
    font-size: 36px;
    margin-bottom: 8px;
  }
  .promo {
    font-size: 18px;
  }
  .why {
    margin-top: 16px;
    font-size: 18px;
    text-align: center;
  }
  .activate {
    margin-top: 24px;
  }
  .name {
    font-size: 16px;
  }
  input {
    width: 30vw;
    min-width: 600px;
    max-width: 100vw;
    line-height: 42px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    padding-left: 20px;
  }
  input:active, :focus {
    outline: none; 
  }
  button {
    background: var(--btn-special);
    color: var(--text-card-color);
    margin: 20px;
    padding: 12px 32px;
    border: none;
    border-radius: 16px;
  }
  button:hover {
    cursor: pointer;
  }
  button:disabled {
    pointer-events: none;
    opacity: .6;
  }
  .get {
    margin-top: 30px;
  }
  a {
    color: var(--text-color);
    text-decoration: underline;
  }
  .skip {
    text-decoration: underline;
    margin-bottom: 20px;
  }
</style>
