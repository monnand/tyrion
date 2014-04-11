cleanFiles(){
	rm -rf src-*
	rm -f sources*.tsv
	rm -f out.tsv 
}

ica(){
	LOG=$1
	./log2tsv.py $LOG > out.tsv
	R CMD BATCH ica.r
}

plotsrc() {
	SRCTSV=$1
	NRCOL=`head -n 1 $SRCTSV | perl -ne "my \\\$n=split; print \\\$n"`
	DIR=`printf "src-%d" $NRCOL`
	mkdir $DIR
	pushd . > /dev/null
	cd $DIR
	cp ../$SRCTSV .
	cp ../cdf.r .
	cp ../density.r .
	for i in `seq 1 $NRCOL`; do
		out=`printf "%d-src.tsv" $i`
		graph=`printf "%d-src-cdf.pdf" $i`
		cat $SRCTSV | perl -ne "my @e=split; print \"\$e[$i-1]\\n\";" > $out
		R CMD BATCH --no-save --no-restore "--args input=\"$out\" output=\"$graph\"" cdf.r

		graph=`printf "%d-src-pdf.pdf" $i`
		R CMD BATCH --no-save --no-restore "--args input=\"$out\" output=\"$graph\"" density.r 

	done
	rm -f cdf.r
	rm -f density.r
	popd > /dev/null
}


cleanFiles
LOGFILE=$1
ica $LOGFILE

for src in sources-*.tsv; do
	echo $src
	plotsrc $src
done
rm -f sources*.tsv

