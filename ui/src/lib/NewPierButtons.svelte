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
      const query = {"app":"boot-new", "data": k }
			fetch($api + '/urbit?urbit_id=' + n.replace(/~/g,''), {
					method: 'POST',
					headers: {'Content-Type': 'application/json'},
					body: JSON.stringify(query)
			})
        .then(d => d.json())
        .then(res => {
          if (res === 200) {
            buttonStatus = 'success'
            setTimeout(window.location.href = "/" + name, 2000)
          } else {
            console.log(res)
            buttonStatus = 'failure'
            setTimeout(()=>buttonStatus = 'standard', 4000)
    }})} else {
      buttonStatus = 'failure'
      setTimeout(()=>buttonStatus = 'standard', 4000)
    }}

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
