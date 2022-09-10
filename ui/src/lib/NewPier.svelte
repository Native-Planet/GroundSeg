<script>
  import { api, isPatp } from '$lib/api'
  import PrimaryButton from '$lib/PrimaryButton.svelte'
  import LinkButton from '$lib/LinkButton.svelte'

  export let name='', key=''

  let buttonStatus = 'standard'
  let prefix = "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
  let suffix = "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

  prefix = prefix.match(/.{1,3}/g)
  suffix = suffix.match(/.{1,3}/g)

  const boot = () => {
    buttonStatus = 'loading'
    const f = new FormData()
    const u = api + "/upload/key"

    if (isPatp(name)) {
      f.append("patp", name.replace(/~/g,''))
      f.append("key", key)
      fetch(u, {method: 'POST',body: f})
        .then(d => d.json())
        .then(res => {
          if (res === 200) {
            buttonStatus = 'success'
            setTimeout(window.location.href = "/" + name, 2000)
          } else {
            buttonStatus = 'failure'
            setTimeout(()=>buttonStatus = 'standard', 4000)
    }})} else {
      buttonStatus = 'failure'
      setTimeout(()=>buttonStatus = 'standard', 4000)
    }}

</script>

<div>

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
    status={(name == '') || (key == '') ? "disabled" : buttonStatus}
    left={false}
  />

</div>

<style>
  div {
    display: flex;
    margin-top: 24px;
  }
</style>
