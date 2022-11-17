<script>
  import { blur } from 'svelte/transition'
  import { api, isPatp } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import LinkButton from '$lib/LinkButton.svelte'

  export let name='', key=''

  let buttonStatus = 'standard'
  let prefix = "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
  let suffix = "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

  prefix = prefix.match(/.{1,3}/g)
  suffix = suffix.match(/.{1,3}/g)

  const isFilled = (n,k,m) => {
    if (n == '') {return true}
    if (k == '') {return true}
    return false
  }

  const boot = () => {
    buttonStatus = 'loading'
    const n = name.trim()
    const k = key.trim()

    if (isPatp(n)) {
      let nr = n.replace(/~/g,'')
      const query = {"app":"boot-new", "data": k }
			fetch($api + '/urbit?urbit_id=' + nr, {
					method: 'POST',
          credentials: 'include',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(query)
			})
        .then(d => d.json())
        .then(res => {
          if (res === 200) {
            handleSuccess(nr)
          } else {
            console.log(res)
            buttonStatus = 'failure'
            setTimeout(()=>buttonStatus = 'standard', 4000)
    }})} else {
      buttonStatus = 'failure'
      setTimeout(()=>buttonStatus = 'standard', 4000)
    }}

  const handleSuccess = n => {
    fetch($api + '/urbit?urbit_id=' + n, {credentials:'include'})
			.then(raw => raw.json())
      .then(res => {
        if (res.name == n) {
          window.location.href = '/' + n
        } else {
          setTimeout(()=> handleSuccess(n), 1000)
        }
      })
			.catch(err => console.log(err))
  }

</script>

<div transition:blur={{duration: 600, amount: 10}}>

  {#if buttonStatus != 'loading'}
    <LinkButton
      text="Cancel"
      src="/"
      disabled={false}
    />
  {/if}

  <PrimaryButton
    on:click={boot}
    standard="Create new pier"
    success="Pier created. Redirecting..."
    failure="Failed. Please check your @p and key"
    loading="Your pier is being created.."
    status={isFilled(name,key) ? "disabled" : buttonStatus}
    left={false}
  />

</div>

<style>
  div {
    display: flex;
    margin-top: 24px;
  }
</style>
