source('ica.r')

plotSrcCdf = function(srcWithHead) {
	srcId=srcWithHead[[1]]
	src=srcWithHead[2:length(srcWithHead)]
	pdf(paste("source", srcId, "pdf", sep="."))
	plot(ecdf(src))
	dev.off()
}
 
inputf='out.tsv'
m = as.matrix(read.table(inputf))
srcs = ica.norm.srcs(m)

srcIds = 1:(dim(srcs)[[2]])
srcsWithHead=rbind(srcIds,srcs)
apply(srcsWithHead, 2, plotSrcCdf)

