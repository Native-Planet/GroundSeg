def start_script():
    return """\
#!/bin/bash

set -eu
# set defaults
amesPort="34343"
httpPort="80"
loom="31"
devMode="False"

# Find the first directory and start urbit with the ship therein
dirnames="*/"
dirs=( $dirnames )
dirname=''${dirnames[0]}

# Patp checker
check_patp() {
    patp="$1"
    pre="dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
    suf="zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"
    [[ "${patp:0:1}" == "~" ]] && patp="${patp:1}"
    patp_arr=(${patp//-/ })

    [[ "${patp:0:3}" == "doz" ]] && return

    if [[ ${#patp} -eq 3 ]]; then
        [[ $suf == *"$patp"* ]] && echo "$patp" && return
    else
        for p in "${patp_arr[@]}"; do
            [[ ${#p} -eq 6 && $pre == *"${p:0:3}"* && $suf == *"${p:3:3}"* ]] || return
        done
        echo "$patp"
    fi
}

# Find a directory with a valid patp
for patp in *; do
    if [[ -d $patp ]]; then
        result=$(echo $(check_patp "$patp"))
        if [[ -n $result ]]; then
          dirname=$result
          break
        fi
    fi
done

# check args
for i in "$@"
do
case $i in
  -p=*|--port=*)
      amesPort="${i#*=}"
      shift
      ;;
   --http-port=*)
      httpPort="${i#*=}"
      shift
      ;;
   --loom=*)
      loom="${i#*=}"
      shift
      ;;
   --dirname=*)
      dirname="${i#*=}"
      shift
      ;;
   --devmode=*)
      devMode="${i#*=}"
      shift
      ;;
esac
done

# If the container is not started with the `-i` flag
# then STDIN will be closed and we need to start
# Urbit/vere with the `-t` flag.
ttyflag=""
if [ ! -t 0 ]; then
echo "Running with no STDIN"
ttyflag="-t"
fi

# Check if there is a keyfile, if so boot a ship with its name, and then remove the key
if [ -e *.key ]; then
    # Get the name of the key
    keynames="*.key"
    keys=( $keynames )
    keyname=''${keys[0]}
    mv $keyname /tmp

    # Boot urbit with the key, exit when done booting
    urbit $ttyflag -w $(basename $keyname .key) -k /tmp/$keyname -p $amesPort -x --http-port $httpPort --loom $loom

    # Remove the keyfile for security
    rm /tmp/$keyname
    rm *.key || true
fi

file="${dirname}/.vere.lock"
if [ -e "$file" ]; then
    content=$(cat "$file")
    if [ "$content" == "1" ]; then
        rm "$file"
        echo "File .vere.lock containing PID 1 has been deleted."
    fi
fi


if [ $devMode == "True" ]; then
    echo "Developer mode: $devMode"
    echo "No logs will display"
    # Run urbit inside a tmux pane (no logs)
    tmux new -d -s urbit "script -q -c 'exec urbit -p $amesPort --http-port $httpPort --loom $loom $dirname' /dev/null"
    tmux_pid=$(tmux list-panes -t urbit -F "#{pane_pid}")
    while kill -0 "$tmux_pid" 2> /dev/null; do
        sleep 3
    done
    tmux kill-session -t urbit
    exit 0
else
    urbit $ttyflag -p $amesPort --http-port $httpPort --loom $loom $dirname
fi
"""
