require('fastICA')
runICAs = function(m) {
	N = dim(m)[2]
	s = sapply(2:N, function(n) {
		ret = fastICA(m, n)
		return(ret$S)
	})
}

dump.ms = function(ms, prefix="sources", sep="-") {
	sapply(ms, function(m) {
	       n = dim(m)[2]
	       f = paste(prefix, n, sep=sep)
	       f = paste(f, "tsv", sep=".")
	       write.table(m, f, row.names=F, col.names=F)
	       return(0);
	})
	return(0);
}

inputf='out.tsv'
m = as.matrix(read.table(inputf))
dump.ms(runICAs(m))
