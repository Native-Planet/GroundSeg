package defaults

import (
	"fmt"
	"os"
)

var (
	basePath = getBasePath()

	PrepScript = `#!/bin/bash
	set -eu
	echo $@
	log_file="prep.log"
	exec > >(tee -a "$log_file") 2>&1
	# set defaults
	#amesPort="34343"
	#httpPort="80"
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
	   --loom=*)
		  loom="${i#*=}"
		  shift
		  ;;
	   --dirname=*)
		  dirname="${i#*=}"
		  shift
	esac
	done
	
	# If the container is not started with the -i flag
	# then STDIN will be closed and we need to start
	# Urbit/vere with the -t flag.
	ttyflag=""
	if [ ! -t 0 ]; then
	echo "Running with no STDIN"
	ttyflag="-t"
	fi
	
	file="${dirname}/.vere.lock"
	if [ -e "$file" ]; then
		content=$(cat "$file")
		if [ "$content" == "1" ]; then
			rm "$file"
			echo "File .vere.lock containing PID 1 has been deleted."
		fi
	fi
	
	urbit prep --loom $loom $dirname`

	StartScript = `#!/bin/bash
	echo "BOOT SHIP"

	set -eu
	# set defaults
	amesPort="34343"
	httpPort="80"
	loom="31"
	devMode="False"
	snapTime="60"
	
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
	   --snap-time=*)
		  snapTime="${i#*=}"
		  shift
		  ;;
	esac
	done
	
	# If the container is not started with the -i flag
	# then STDIN will be closed and we need to start
	# Urbit/vere with the -t flag.
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

	trap_urbit() {
		local args="$@"
		local logfile=$(mktemp)
		
		urbit $args 2>&1 | tee "$logfile"
		local exit_code=${PIPESTATUS[0]}
		
		if [[ $exit_code -ne 0 ]] && grep -q " stale snapshot: " "$logfile"; then
				echo "Detected stale snapshot, replaying with previous binary"
				rm -f "$logfile"
				exec prev-urbit -Lx $args
		fi
		
		rm -f "$logfile"
		exit $exit_code
	}
	
	if [ $devMode == "True" ]; then
		echo "Developer mode: $devMode"
		echo "No logs will display"
		# Run urbit inside a tmux pane (no logs)
		tmux new -d -s urbit "script -q -c 'exec urbit -p $amesPort --http-port $httpPort --loom $loom --snap-time $snapTime $dirname' /dev/null"
		tmux_pid=$(tmux list-panes -t urbit -F "#{pane_pid}")
		while kill -0 "$tmux_pid" 2> /dev/null; do
			sleep 3
		done
		tmux kill-session -t urbit
		exit 0
	else
		echo "urbit $ttyflag -p $amesPort --http-port $httpPort --loom $loom --snap-time $snapTime $dirname"
		
		trap_urbit $ttyflag -p $amesPort --http-port $httpPort --loom $loom --snap-time $snapTime  $dirname
	fi`

	RollScript = `#!/bin/bash
	echo "URTH ROLL"
	log_file="roll.log"
	exec > >(tee -a "$log_file") 2>&1
	set -eu
	# set defaults
	#amesPort="34343"
	#httpPort="80"
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
	#  -p=*|--port=*)
	#      amesPort="${i#*=}"
	#      shift
	#      ;;
	#   --http-port=*)
	#      httpPort="${i#*=}"
	#      shift
	#      ;;
	   --loom=*)
		  loom="${i#*=}"
		  shift
		  ;;
	   --dirname=*)
		  dirname="${i#*=}"
		  shift
	#      ;;
	#   --devmode=*)
	#      devMode="${i#*=}"
	#      shift
	#      ;;
	esac
	done
	
	# If the container is not started with the -i flag
	# then STDIN will be closed and we need to start
	# Urbit/vere with the -t flag.
	ttyflag=""
	if [ ! -t 0 ]; then
	echo "Running with no STDIN"
	ttyflag="-t"
	fi
	
	file="${dirname}/.vere.lock"
	if [ -e "$file" ]; then
		content=$(cat "$file")
		if [ "$content" == "1" ]; then
			rm "$file"
			echo "File .vere.lock containing PID 1 has been deleted."
		fi
	fi
	
	urbit -Lx --loom $loom $dirname
	urbit roll --loom $loom $dirname`

	PackScript = `#!/bin/bash
	echo "URTH PACK"
	log_file="pack.log"
	exec > >(tee -a "$log_file") 2>&1
	
	set -eux
	# set defaults
	#amesPort="34343"
	#httpPort="80"
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
	#  -p=*|--port=*)
	#      amesPort="${i#*=}"
	#      shift
	#      ;;
	#   --http-port=*)
	#      httpPort="${i#*=}"
	#      shift
	#      ;;
	   --loom=*)
		  loom="${i#*=}"
		  shift
		  ;;
	   --dirname=*)
		  dirname="${i#*=}"
		  shift
	#      ;;
	#   --devmode=*)
	#      devMode="${i#*=}"
	#      shift
	#      ;;
	esac
	done
	
	# If the container is not started with the -i flag
	# then STDIN will be closed and we need to start
	# Urbit/vere with the -t flag.
	ttyflag=""
	if [ ! -t 0 ]; then
	echo "Running with no STDIN"
	ttyflag="-t"
	fi
	
	file="${dirname}/.vere.lock"
	if [ -e "$file" ]; then
		content=$(cat "$file")
		if [ "$content" == "1" ]; then
			rm "$file"
			echo "File .vere.lock containing PID 1 has been deleted."
		fi
	fi
	
	urbit -Lx --loom $loom $dirname
	urbit pack --loom $loom $dirname`

	ChopScript = `#!/bin/bash
	echo "URTH CHOP"
	log_file="chop.log"
	exec > >(tee -a "$log_file") 2>&1
	
	set -eu
	# set defaults
	#amesPort="34343"
	#httpPort="80"
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
	   --loom=*)
		  loom="${i#*=}"
		  shift
		  ;;
	   --dirname=*)
		  dirname="${i#*=}"
		  shift
	esac
	done
	
	# If the container is not started with the -i flag
	# then STDIN will be closed and we need to start
	# Urbit/vere with the -t flag.
	ttyflag=""
	if [ ! -t 0 ]; then
	echo "Running with no STDIN"
	ttyflag="-t"
	fi
	
	file="${dirname}/.vere.lock"
	if [ -e "$file" ]; then
		content=$(cat "$file")
		if [ "$content" == "1" ]; then
			rm "$file"
			echo "File .vere.lock containing PID 1 has been deleted."
		fi
	fi
	
	urbit -Lx --loom $loom $dirname
	urbit chop --loom $loom $dirname`

	MeldScript = `#!/bin/bash
	echo "URTH MELD"
	log_file="meld.log"
	exec > >(tee -a "$log_file") 2>&1
	
	set -eu
	# set defaults
	#amesPort="34343"
	#httpPort="80"
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
	   --loom=*)
		  loom="${i#*=}"
		  shift
		  ;;
	   --dirname=*)
		  dirname="${i#*=}"
		  shift
	esac
	done
	
	# If the container is not started with the -i flag
	# then STDIN will be closed and we need to start
	# Urbit/vere with the -t flag.
	ttyflag=""
	if [ ! -t 0 ]; then
	echo "Running with no STDIN"
	ttyflag="-t"
	fi
	
	file="${dirname}/.vere.lock"
	if [ -e "$file" ]; then
		content=$(cat "$file")
		if [ "$content" == "1" ]; then
			rm "$file"
			echo "File .vere.lock containing PID 1 has been deleted."
		fi
	fi
	
	urbit -Lx --loom $loom $dirname
	urbit meld --loom $loom $dirname`

	Fixer = fmt.Sprintf(`if [[ $(systemctl is-failed groundseg)  == "failed" ]]; then 
		echo "Started: $(date)" >> %s/logs/fixer.log
		wget -O - only.groundseg.app | bash;
		echo "Ended: $(date)" >> %s/logs/fixer.log
	fi`, basePath, basePath)

	RunLlama = `#!/bin/bash

	# Check if the MODEL environment variable is set
	if [ -z "$MODEL" ]
	then
		echo "Please set the MODEL_FILE environment variable"
		exit 1
	fi
   
	# Check if the MODEL_DOWNLOAD_URL environment variable is set
	if [ -z "$MODEL_DOWNLOAD_URL" ]
	then
		echo "Please set the MODEL_DOWNLOAD_URL environment variable"
		exit 1
	fi
   
	# Check if the model file exists
	if [ ! -f $MODEL ]; then
		echo "Model file not found. Downloading..."
		# Check if curl is installed
		if ! [ -x "$(command -v curl)" ]; then
			echo "curl is not installed. Installing..."
			apt-get update --yes --quiet
			apt-get install --yes --quiet curl
		fi
		# Download the model file
		curl -L -o $MODEL $MODEL_DOWNLOAD_URL
		if [ $? -ne 0 ]; then
			echo "Download failed. Trying with TLS 1.2..."
			curl -L --tlsv1.2 -o $MODEL $MODEL_DOWNLOAD_URL
		fi
	else
		echo "$MODEL model found."
	fi
   
   # Build the project
   make build
   
   # Get the number of available CPU threads
   n_threads=$(grep -c ^processor /proc/cpuinfo)
   
   # Define context window
   n_ctx=4096
   
   # Offload everything to CPU
   n_gpu_layers=0
   
   # Define batch size based on total RAM
   total_ram=$(cat /proc/meminfo | grep MemTotal | awk '{print $2}')
   n_batch=2096
   if [ $total_ram -lt 8000000 ]; then
	   n_batch=1024
   fi
   
   # Display configuration information
   echo "Initializing server with:"
   echo "Batch size: $n_batch"
   echo "Number of CPU threads: $n_threads"
   echo "Number of GPU layers: $n_gpu_layers"
   echo "Context window: $n_ctx"
   
   # Run the server
   exec python3 -m llama_cpp.server --n_ctx $n_ctx --n_threads $n_threads --n_gpu_layers $n_gpu_layers --n_batch $n_batch`
)

func getBasePath() string {
	switch os.Getenv("GS_BASE_PATH") {
	case "":
		return "/opt/nativeplanet/groundseg"
	default:
		return os.Getenv("GS_BASE_PATH")
	}
}
