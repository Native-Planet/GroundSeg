<script>
  import { scale } from 'svelte/transition'
  import { onMount, onDestroy } from 'svelte'
  import { api, updateState } from '$lib/api'
  import { page } from '$app/stores'

  import Fa from 'svelte-fa'
  import { faLock } from '@fortawesome/free-solid-svg-icons'

	import Card from '$lib/Card.svelte'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import EyeButton from '$lib/EyeButton.svelte'

  export let data
  updateState(data)

  let inView = false
  let showLogin = false
  let pwdView = false
  let loginPassword = ''
  let buttonStatus = 'standard'
  let pubKey = ''
  let unlocked = true
  let remainder = 0
  let hours = 0
  let minutes = 0
  let seconds = 0

	onDestroy(()=> inView = false)
  onMount(()=> {
    if (data['status'] == 200) {
      console.log("logged in")
      window.location.href = "/"
    } else if (data['status'] == 'setup') {
      window.location.href = "/setup"
    } else {
      console.log(data['status'])
    }
    isLocked()
    inView = true
    getLoginKey()
  })

  const getLoginKey = () => {
    if (inView && ($page.url.pathname == "/login")) {
      fetch($api + '/login/key')
      .then(r => r.json())
        .then(d => {
          pubKey = d
        })
      setTimeout(getLoginKey, 30000)
  }}

  let count = 0

  const isLocked = () => {
    if ($page.url.pathname == "/login") {
      fetch($api + '/login/status')
      .then(r => r.json())
      .then(d => {
        if (d != 400) {
          remainder = d.remainder
          if (d.locked) {
            startCountdown()
          }
        }
      })
      .catch(e => console.log(e))
    }
  }

  const startCountdown = () => {
    const countdown = setInterval(() => {
      if (remainder <= 0) {
        unlocked = true
        clearInterval(countdown)
        isLocked()
      } else {
        hours = Math.floor(remainder / 3600)
        minutes = Math.floor((remainder % 3600) / 60)
        seconds = Math.floor(remainder % 60)
        unlocked = false
        remainder--
      }
    }, 1000)
  }

  const togglePwdView = () => {
    pwdView = !pwdView
    document.querySelector('#login-password').type = pwdView ? 'text' : 'password'
  }

  const handleLogin = async () => {
    buttonStatus = 'loading'
    const enc_pwd = await encryptPassword()

    fetch($api + '/login', {
			method: 'POST',
      headers: {"Content-Type" : "application/json"},
      credentials : "include",
			body: JSON.stringify({'password':enc_pwd})
	  })
      .then(r => r.json())
      .then(d => { 
        if (d == 200) {
          buttonStatus = 'success'
          setTimeout(()=> window.location.href = '/', 1000)
        } else {
          console.log(d)
          buttonStatus = 'failure'
          setTimeout(()=> isLocked(), 1000)
          setTimeout(()=> {
            buttonStatus = 'standard'
            loginPassword = ''
          }, 2000)
        }
    })
  }

  const encryptPassword = async () => {
    const encrypt = new JSEncrypt({ default_key_size: 2048 })
    encrypt.setPublicKey(pubKey)
    const encrypted = await encrypt.encrypt(loginPassword.trim())
    return encrypted
  }

</script>

<svelte:head><script src="/jsencrypt.min.js"></script></svelte:head>

{#if inView}
  <Card width="640px" bgColor={unlocked ? "" : "#3F00008D"}>
    <div class="main-wrapper">
      <div class="opened-wrapper" in:scale={{delay:400, duration: 120}}>
        <img src="/npfull.svg" alt="Native Planet Logo" />
        {#if unlocked}
          <div class="login-wrapper">
            <input
              id="login-password"
              bind:value={loginPassword}
              class="login-password"
              type="password"
              placeholder='Password'
              disabled={buttonStatus != 'standard'}
              on:keydown={e => {
                if (e.key === 'Enter') {
                  handleLogin()
                }
            }}>
          </div>
          <PrimaryButton
            top=24
            left={false}
            standard="Login"
            success="Login successful!"
            failure="Login failed"
            loading="Logging you in.."
            status={(loginPassword.length > 0) && (pubKey.length > 0) ? buttonStatus : 'disabled'}
            on:click={handleLogin}
          />
        {:else}
          <div class="locked">
            <div class="locked-icon"><Fa icon={faLock} size="4x" /></div>
            <div class="locked-text">{hours > 0 ? hours + " HOURS" : ""} {minutes > 0 ? minutes + " MINUTES" : ""} {seconds} SECONDS</div>
          </div>
        {/if}
      </div>
    </div>
  </Card>
{/if}

<style>
  .main-wrapper {
    text-align: center;
    margin: 40px 80px;
  }
  .login-wrapper {
    margin-top: 24px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .login-password {
    font-size: 12px;
    padding: 8px;
    background: #ffffff4d;
    border-radius: 6px;
    flex: 1;
    border: none;
    font-family: inherit;
    color: inherit;
  }
  input {
    text-align: center;
  }
  input:focus {
    outline: none;
  }
  input:disabled {
    opacity: 0.4;
  }
  input::placeholder {
    color: inherit;
    opacity: .6;
  }
  .locked {
    text-align: center;
    margin-top: 30px;
    line-height: 40px;
  }
  .locked-icon {
    color: red;
  }
  .locked-text {
    margin-top: 30px;
    font-size: 24px;
  }
</style>
