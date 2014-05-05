library('fastICA')

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

ica.norm.srcs = function(obv) {
	n = dim(obv)[2]
	icares = fastICA(obv, n)
	srcs = icares$S
	srcs = apply(srcs, 2, negate)
	srcs = apply(srcs, 2, normalize)
	return(srcs)
}

tailDiffCurry = function(a, b) {
	ret = function(src) {
		p = quantile(ecdf(src), a)
		q = quantile(ecdf(src), b)
		return(abs(q-p))
	}
	return(ret)
}

