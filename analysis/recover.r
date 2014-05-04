library('fastICA')
library("parallel")
library('ggplot2')
library('reshape2')

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

normalize = function(data) {
	data = data + abs(min(data))
	data = data / abs(max(data))
	return(data)
}

negate = function(data) {
	m = mean(normalize(data))
	if (m > 0.5) {
		return(-data)
	}
	return(data)
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

ica.norm.srcs = function(obv) {
	n = dim(obv)[2]
	icares = fastICA(obv, n)
	srcs = icares$S
	negsrcs = apply(srcs, 2, negate)
	normsrcs = apply(negsrcs, 2, normalize)
	return(normsrcs)
}
 
percentile.curry = function(percent) {
	ret = function(data) {
		return(quantile(ecdf(data), percent))
	}
	return(ret)
}

multi.percentiles.curry = function(percents) {
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

multi.indices.curry = function(idcs) {
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
 
predictSinceBeginning = function(obv, step, reduceFuncWithinSrc, reduceFuncBetweenSrcs=max, col.names=0, alpha=0.8, sla_percentage=0.999, sla_time=14000000000) {
	n = dim(obv)[[1]]
	ends = seq(step,n,by=step)
	em = matrix(ends, length(ends), 1)
	print(em)
	ret = apply(em, 1, function(e) {
		end = e[[1]]
		print(paste("Caltulating from 1 to", end))
		s = sample(obv, 1, 1, end)
		normsrcs = ica.norm.srcs(s)
		recsrcs = apply(normsrcs, 2, function(src) {
			return(reduceFuncWithinSrc(recover(src, s, alpha, sla_percentage, sla_time)))
		})
		r = reduceFuncBetweenSrcs(recsrcs)
		print(paste("Caltulated from 1 to", end, ": ", r))
		return(r)
	})
	x = data.frame(ends,t(ret))
	if (length(col.names) != dim(x)[[2]] - 1) {
		col.names = 1:(dim(x)[[2]] - 1)
	}
	names(x) = c("time", col.names)
	return(x)
}
 
predictWithinWindow = function(obv, windowSize, reduceFuncWithinSrc, reduceFuncBetweenSrcs=max, col.names=0, alpha=0.8, sla_percentage=0.999, sla_time=14000000000) {
	n = dim(obv)[[1]]
	ends = seq(windowSize,n,by=windowSize)
	em = matrix(ends, length(ends), 1)
	print(em)
	ret = apply(em, 1, function(e) {
		end = e[[1]]
		start = end-windowSize
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
	if (length(col.names) != dim(x)[[2]] - 1) {
		col.names = 1:(dim(x)[[2]] - 1)
	}
	names(x) = c("time", col.names)
	return(x)
}

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


window = function() {
	obv=as.matrix(read.table('out.tsv'))
	#s=sample(obv,1,1,1000)
	s=obv
	step=500
	src.reduce.percentiles = c(0.5, 0.8, 0.9, 1.0)
	#src.idcs = 1:10
	src.idcs = c(1, 6, 9)
	col.names = sapply(src.reduce.percentiles, function(p) {
		paste("Predicted 99.99%tile.", p*100, "%tile source")
	})
	idcs.col.names = sapply(src.idcs, function(p) {
		paste("Predicted 99.99%tile.", p*10, "%tile source")
	})
	rs=predictWithinWindow(s, step, percentile.curry(0.9999), multi.indices.curry(src.idcs), idcs.col.names, alpha=0.0)

	rd=as.vector(reduceDataWithinWindow(s, step, percentile.curry(0.9999))[,2])
	rs=appendColToDataFrame(rs, rd, "Observed 99.99%tile")

	rd=as.vector(reduceDataWithinWindow(s, step, max)[,2])
	rs=appendColToDataFrame(rs, rd, "Observed max")

	melted = melt(rs, id.vars="time")
	ggplot(data=melted, aes(x=time, y=value, group=variable, color=variable)) + geom_line()
	ggsave(file="window.pdf", width=15, height=7)
}

main = function() {
	obv=as.matrix(read.table('out.tsv'))
	#s=sample(obv,1,1,1000)
	s=obv
	step=200
	src.reduce.percentiles = c(0.5, 0.8, 0.9, 1.0)
	#src.idcs = 1:10
	src.idcs = c(1, 6, 9)
	col.names = sapply(src.reduce.percentiles, function(p) {
		paste("Predicted 99.99%tile.", p*100, "%tile source")
	})
	idcs.col.names = sapply(src.idcs, function(p) {
		paste("Predicted 99.99%tile.", p*10, "%tile source")
	})
	rs=predictSinceBeginning(s, step, percentile.curry(0.9999), multi.indices.curry(src.idcs), idcs.col.names, alpha=0.0)

	rd=as.vector(reduceDataSinceBeginning(s, step, percentile.curry(0.9999))[,2])
	rs=appendColToDataFrame(rs, rd, "Observed 99.99%tile")

	rd=as.vector(reduceDataSinceBeginning(s, step, max)[,2])
	rs=appendColToDataFrame(rs, rd, "Observed max")

	melted = melt(rs, id.vars="time")
	ggplot(data=melted, aes(x=time, y=value, group=variable, color=variable)) + geom_line()
	ggsave(file="test.pdf", width=15, height=7)
}
