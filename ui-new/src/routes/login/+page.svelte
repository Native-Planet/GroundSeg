<script>
  let wide = true
</script>

<div class="container {wide ? "wide" : "slim"}">
  <div class="title">LOGIN</div>
  <input type="password"/>
  <button>Submit</button>
</div>

<style>
  .container {
    display: flex;
    gap: 20px;
    background-color: var(--bg-base);
    color: var(--text-color);
    margin: auto;
    border-radius: 16px;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }
  .wide {
    width: 992px;
    height: calc(100vh - 240px);
    margin-top: 120px;
    max-width: 98vw;
  }
  .slim {
    width: 100vw;
  }
  .title {
    font-size: 60px;
    font-family: var(--title-font);
  }
  input {
    width: 50%;
    line-height: 42px;
    border: solid 2px var(--btn-secondary);
    border-radius: 16px;
    background: none;
    text-align: center;
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
</style>
