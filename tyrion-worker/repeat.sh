N=100
sec=30
while getopts "n:s:" opt; do
	case $opt in
		n)
			N=$OPTARG
			;;
		s)
			sec=$OPTARG
			;;
	esac
done

shift $((OPTIND-1))

for i in `seq 1 $N`; do
	for file in "$@"; do
		./tyrion-worker -json $file
	done
	sleep $sec
done
