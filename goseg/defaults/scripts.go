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
			for p in "${patp_arr[@]}"; dofa
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

		# probe for stale snapshot with a quick exit
		local probe_log=$(mktemp)
		echo "Probing pier with urbit -Lx..."
		if urbit -Lx $ttyflag --loom $loom $dirname >"$probe_log" 2>&1; then
			probe_exit=0
		else
			probe_exit=$?
		fi

		if [[ $probe_exit -ne 0 ]] && grep -q "stale snapshot" "$probe_log"; then
			echo "Detected stale snapshot, attempting vere migration rollback"
			rm -f "$probe_log"

			# Detect architecture
			export device_arch=$(uname -m)
			export asset_name=""
			if [[ $device_arch == "aarch64" ]]; then
				asset_name="linux-aarch64.tgz"
			elif [[ $device_arch == "x86_64" ]]; then
				asset_name="linux-x86_64.tgz"
			else
				echo "Unsupported architecture: $device_arch"
				exit 1
			fi

			export version_output=$(urbit -R 2>&1 || true)
			export raw_version=$(echo "$version_output" | awk 'tolower($1) == "urbit" {print $2; exit}')
			if [[ -z "$raw_version" ]]; then
				echo "Cannot determine current vere version from 'urbit -R'"
				exec prev-urbit -Lx $ttyflag $args
			fi
			# normalize to release tag format
			export cleaned=$(echo "$raw_version" | sed 's/^vere-//; s/^v//')
			export ver_num=$(echo "$cleaned" | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?')
			if [[ -z "$ver_num" ]]; then
				echo "Cannot parse version number from: $raw_version"
				exec prev-urbit -Lx $ttyflag $args
			fi
			export current_tag="vere-v${ver_num}"
			echo "Detected current vere version: $current_tag"

			# fetch releases from GitHub
			releases=$(curl -sL "https://api.github.com/repos/urbit/vere/releases?per_page=100")
			# build ordered list of release tags
			export -a tags_array=()
			export current_idx=-1
			export idx=0
			while IFS= read -r tag; do
				tags_array+=("$tag")
				if [[ "$tag" == "$current_tag" ]]; then
					current_idx=$idx
				fi
				idx=$((idx + 1))
			done < <(echo "$releases" | jq -r '.[] | select(.draft == false and .prerelease == false) | .tag_name')

			if [[ $current_idx -eq -1 ]]; then
				# not an exact match, prob prerelease
				# Find insertion point: first release with version <= our base
				export base_major=$(echo "$ver_num" | cut -d. -f1)
				export base_minor=$(echo "$ver_num" | cut -d. -f2)
				export insert_idx=${#tags_array[@]}
				for (( j=0; j < ${#tags_array[@]}; j++ )); do
					export tv=$(echo "${tags_array[$j]}" | sed 's/^vere-v//; s/-.*//')
					export tm=$(echo "$tv" | cut -d. -f1)
					export tn=$(echo "$tv" | cut -d. -f2)
					if (( tm < base_major || (tm == base_major && tn <= base_minor) )); then
						insert_idx=$j
						break
					fi
				done
				# set current_idx just before that release (virtual position)
				current_idx=$((insert_idx - 1))
				echo "Test build detected, will migrate through releases from index $insert_idx"
			fi

			export tmpdir=$(mktemp -d)

			# Download and cache a vere binary for a given release tag.
			fetch_vere() {
				export tag="$1"
				export bindir="$tmpdir/$tag"
				binpath=""
				mkdir -p "$bindir"
				export url=$(echo "$releases" | jq -r --arg tag "$tag" --arg name "$asset_name" '
					.[] | select(.tag_name == $tag)
						| .assets[]?
						| select(.name == $name)
						| .browser_download_url
					' | head -n 1)
				if [[ -z "$url" || "$url" == "null" ]]; then
					return
				fi
				if ! curl -sL "$url" | tar xz -C "$bindir" 2>/dev/null; then
					return
				fi
				binpath=$(find "$bindir" -maxdepth 1 -type f -name 'vere-*' | head -1)
				if [[ -n "$binpath" ]]; then
					chmod +x "$binpath"
				fi
			}

			# Iterate backwards to find a version that works >= vere-v3.5
			export working_idx=-1
			for (( i = current_idx + 1; i < ${#tags_array[@]}; i++ )); do
				export tag="${tags_array[$i]}"
				# Stop if we've gone past the floor version
				export tv=$(echo "$tag" | sed 's/^vere-v//; s/-.*//')
				export tm=$(echo "$tv" | cut -d. -f1)
				export tn=$(echo "$tv" | cut -d. -f2)
				if (( tm < 3 || (tm == 3 && tn < 5) )); then
					echo "Reached floor version (vere-v3.5), stopping backward search"
					break
				fi
				export binpath=""
				fetch_vere "$tag"
				if [[ -z "$binpath" ]]; then
					echo "No binary for $tag on $device_arch, skipping"
					continue
				fi
				echo "Trying older version $tag..."
				if $binpath -Lx $ttyflag --loom $loom $dirname; then
					echo "Version $tag succeeded"
					working_idx=$i
					break
				else
					echo "Version $tag failed (exit $?), trying older..."
				fi
			done

			if [[ $working_idx -eq -1 ]]; then
				echo "No working older version found"
				rm -rf "$tmpdir"
				exit 1
			fi

			# Iterate forward from working version back to current
			for (( i = working_idx - 1; i > current_idx; i-- )); do
				export tag="${tags_array[$i]}"
				export binpath=""
				fetch_vere "$tag"
				if [[ -z "$binpath" ]]; then
					echo "No binary for $tag, skipping forward step"
					continue
				fi
				echo "Migrating forward to $tag..."
				if ! $binpath -Lx $ttyflag --loom $loom $dirname; then
					echo "Warning: migration step $tag failed (exit $?), continuing..."
				fi
			done

			echo "Migrating to current version $current_tag..."
			urbit -Lx $ttyflag --loom $loom $dirname

			rm -rf "$tmpdir"
			echo "Migration complete"
			exec urbit $args
		fi

		rm -f "$probe_log"
		exec urbit $args
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
	
	urbit -Lx $ttyflag --loom $loom $dirname
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
	
	urbit -Lx $ttyflag --loom $loom $dirname
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
	
	urbit -Lx $ttyflag --loom $loom $dirname
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
	
	urbit -Lx $ttyflag --loom $loom $dirname
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
