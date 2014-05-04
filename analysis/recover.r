library('fastICA')
library("parallel")
library('ggplot2')
library('reshape2')

source('ica.r')

sample = function(input, period=6, start=100, end=300) {
	total = dim(input)[1]
	idx = 1:total
	filter = c(T, rep(F, period - 1))
	if (end > total) {
		end = total
	}
	ret=input[start:end,][filter,]
	return(ret)
}

recover.with.two.points = function(src, obv, pt1, pt2) {
	# Equations:
	# We know that
	# a * x_norm + b = x
	# and we want to calculate a and b
	#
	# let (x1, y1) = pt1; (x2, y2) = pt2
	# let p = percentile(x_norm, y1);
	# let q = percentile(x_norm, y2);
	# a * p + b = x1
	# a * q + b = x2
	#
	# =>
	# a = (x1 - x2)/(p - q)
	# b = x1 - a * p

	x1 = pt1[[1]]
	x2 = pt2[[1]]
	y1 = pt1[[2]]
	y2 = pt2[[2]]

	p = quantile(ecdf(src), y1)
	q = quantile(ecdf(src), y2)

	a = (x1 - x2)/(p-q)
	b = x1 - a * p
	return(src * a + b)
}

recover = function(src, obv, alpha=0.8, sla_percentage = 0.99, sla_time = 3300000000) {
	# Equations:
	# We know that
	# a * x_norm + b = x
	# and we want to calculate a and b
	#
	# let x_sla = percentile(x_norm, sla_percentage)
	# a * x_sla + b= sla_time
	# b = alpha * min(obv)
	#
	# =>
	#
	# b = alpha * min(obv)
	# a = (sla_time - b) / x_sla
	min_obv = min(obv)
	shift = min_obv * alpha
	x_sla = quantile(ecdf(src), sla_percentage)
	scale = (sla_time - shift) / x_sla

	ret=src * scale + shift
	return(ret)
}

# ica.then.recover = function(obv, alpha=0.8, sla_percentage=0.99, sla_time=3300000000) {
# 	n = dim(obv)[2]
# 	icares = fastICA(obv, n)
# 	srcs = icares$S
# 	negsrcs = apply(srcs, 2, negate)
# 	normsrcs = apply(negsrcs, 2, normalize)
# 
# 	recsrcs = apply(normsrcs, 2, function(src) {
# 			return(recover(src, obv, alpha, sla_percentage, sla_time))
# 		})
# 	return(recsrcs)
# }

percentile.curry = function(percent) {
	ret = function(data) {
		return(quantile(ecdf(data), percent))
	}
	return(ret)
}

multiPercentilesCurry = function(percents) {
	ret = function(data) {
		r = sapply(percents, function(percent) {
			if (percent == 1.0) {
				return(max(data))
			}
			return(as.vector(quantile(ecdf(data), percent)))
		})
		return(r)
	}
	return(ret)
}

multiIndicesCurry = function(idcs) {
	ret = function(data) {
		r = sapply(idcs, function(idx) {
			   return(sort(as.vector(data))[[idx]])
		})
		return(r)
	}
	return(ret)
}

# try.alphas = function(obv, reducefn, srcs.reduce.fn, alphas=seq(0,1,by=0.01), sla_percentage=0.999, sla_time=1400000000) {
# 	normsrcs = ica.norm.srcs(obv)
# 
# 	ret=sapply(alphas, function(alpha) {
# 		recsrcs = apply(normsrcs, 2, function(src) {
# 			return(reducefn(recover(src, obv, alpha, sla_percentage, sla_time)))
# 		})
# 		print(length(recsrcs))
# 		return(srcs.reduce.fn(recsrcs))
# 	})
# }

# cbind function but always cut/pad y to the same length as x before doing cbind
cbindWithPadding = function(x, y) {
	if (length(x) > length(y)) {
		padding = rep(y[[length(y)]], length(x) - length(y))
		y = c(y, padding)
	} else if (length(x) < length(y)) {
		y = y[1:length(x)]
	}
	return(cbind(x,y))
}


predictWithinRange = function(obv, param_matrix, reduceFuncWithinSrc, reduceFuncBetweenSrcs=max, colNames=0, alpha=0.2, sla_percentage=0.999) {
	ends=param_matrix[,2]
	ret = apply(param_matrix, 1, function(e) {
		start = e[[1]]
		end = e[[2]]
		sla_time = e[[3]]
		print(paste("Caltulating from", start, "to", end))
		s = sample(obv, 1, start, end)
		normsrcs = ica.norm.srcs(s)
		recsrcs = apply(normsrcs, 2, function(src) {
			return(reduceFuncWithinSrc(recover(src, s, alpha, sla_percentage, sla_time)))
		})
		r = reduceFuncBetweenSrcs(recsrcs)
		print(paste("Caltulated from", start, "to", end, ": ", r))
		return(r)
	})
	x = data.frame(ends,t(ret))
	if (length(colNames) != dim(x)[[2]] - 1) {
		colNames = 1:(dim(x)[[2]] - 1)
	}
	names(x) = c("time", colNames)
	return(x)

}
 
predictWithinWindow = function(obv, windowSize, reduceFuncWithinSrc, reduceFuncBetweenSrcs=max, colNames=0, alpha=0.8, sla_percentage=0.999, sla_times=14000000000) {
	n = dim(obv)[[1]]
	ends = seq(windowSize,n,by=windowSize)
	starts = seq(1, ends[[length(ends)]], by=windowSize)
	em = cbindWithPadding(ends, sla_times)
	em = cbind(starts,em)
	return(predictWithinRange(obv, em, reduceFuncWithinSrc, reduceFuncBetweenSrcs, colNames=colNames, alpha=alpha, sla_percentage=sla_percentage))
}

predictSinceBeginning = function(obv, step, reduceFuncWithinSrc, reduceFuncBetweenSrcs=max, colNames=0, alpha=0.8, sla_percentage=0.999, sla_times=14000000000) {
	n = dim(obv)[[1]]
	ends = seq(windowSize,n,by=windowSize)
	starts = rep(1, length(ends))
	em = cbindWithPadding(ends, sla_times)
	em = cbind(starts,em)
	return(predictWithinRange(obv, em, reduceFuncWithinSrc, reduceFuncBetweenSrcs, colNames=colNames, alpha=alpha, sla_percentage=sla_percentage))
}
 
# predictSinceBeginning = function(obv, step, reduceFuncWithinSrc, reduceFuncBetweenSrcs=max, col.names=0, alpha=0.8, sla_percentage=0.999, sla_times=14000000000) {
# 	n = dim(obv)[[1]]
# 	ends = seq(step,n,by=step)
# 	em = cbindWithPadding(ends, sla_times)
# 	print(em)
# 	ret = apply(em, 1, function(e) {
# 		end = e[[1]]
# 		sla_time = e[[2]]
# 		print(paste("Caltulating from 1 to", end))
# 		s = sample(obv, 1, 1, end)
# 		normsrcs = ica.norm.srcs(s)
# 		recsrcs = apply(normsrcs, 2, function(src) {
# 			return(reduceFuncWithinSrc(recover(src, s, alpha, sla_percentage, sla_time)))
# 		})
# 		r = reduceFuncBetweenSrcs(recsrcs)
# 		print(paste("Caltulated from 1 to", end, ": ", r))
# 		return(r)
# 	})
# 	x = data.frame(ends,t(ret))
# 	if (length(col.names) != dim(x)[[2]] - 1) {
# 		col.names = 1:(dim(x)[[2]] - 1)
# 	}
# 	names(x) = c("time", col.names)
# 	return(x)
# }

reduceDataSinceBeginning = function(obv, step, reducefn) {
	n = dim(obv)[[1]]
	ends = seq(step,n,by=step)
	ret = sapply(ends, function(end) {
		     s = sample(obv, 1, 1, end)
		     r = reducefn(s)
		     return(r)
	})
	return(cbind(ends, ret))
}

reduceDataWithinWindow = function(obv, windowSize, reducefn) {
	n = dim(obv)[[1]]
	ends = seq(windowSize,n,by=windowSize)
	ret = sapply(ends, function(end) {
		     start=end-windowSize
		     s = sample(obv, 1, start, end)
		     r = reducefn(s)
		     return(r)
	})
	return(cbind(ends, ret))

}

appendColToDataFrame = function(rs, rd, name) {
	col.names=c(names(rs), name)
	rs=data.frame(rs,rd)
	names(rs)=col.names
	return(rs)
}

icaAnalysis = function(predictFunc, reduceDataFunc, reduceFuncForSLA, step=200, input='out.tsv', output='window.pdf') {
	percentile=0.9999
	alpha=0.2
	obv=as.matrix(read.table(input))
	#s=sample(obv,1,1,1000)
	s=obv
	srcIndcs = c(1, 100)
	sla_percentage=0.999
	sla_init_time=14000000000
	colNames = sapply(srcIndcs, function(p) {
		paste("Predicted 99.99%tile.", p, "%tile source")
	})
	sla_times=as.vector(reduceFuncForSLA(s, step, percentile.curry(sla_percentage))[,2])
	sla_times=c(sla_init_time, sla_times)
	print(sla_times)

	rs=predictFunc(
		       s,
		       step,
		       percentile.curry(percentile),
		       multiIndicesCurry(srcIndcs),
		       colNames=colNames,
		       alpha=alpha,
		       sla_percentage=sla_percentage,
		       sla_times=sla_times
		       )

	rd=as.vector(reduceFuncForSLA(s, step, percentile.curry(sla_percentage))[,2])
	rs=appendColToDataFrame(rs, rd, paste("Observed SLA time"))

	rd=as.vector(reduceDataFunc(s, step, percentile.curry(sla_percentage))[,2])
	rs=appendColToDataFrame(rs, rd, paste("Observed", sla_percentage,"%tile"))

	rd=as.vector(reduceDataFunc(s, step, percentile.curry(percentile))[,2])
	rs=appendColToDataFrame(rs, rd, paste("Observed", percentile,"%tile"))

	rd=as.vector(reduceDataFunc(s, step, max)[,2])
	rs=appendColToDataFrame(rs, rd, "Observed max")

	rd=as.vector(reduceDataFunc(s, step, min)[,2])
	rs=appendColToDataFrame(rs, rd, "Observed min")

	melted = melt(rs, id.vars="time")
	ggplot(data=melted, aes(x=time, y=value, group=variable, color=variable)) + geom_line()
	ggsave(file=output, width=15, height=7)
}

main = function() {
	icaAnalysis(predictWithinWindow, reduceDataWithinWindow, reduceDataSinceBeginning, step=200, input='out.tsv', output='window-vary-sla-all-history.pdf')
	icaAnalysis(predictWithinWindow, reduceDataWithinWindow, reduceDataWithinWindow, step=200, input='out.tsv', output='window-vary-sla-window-history.pdf')
}
