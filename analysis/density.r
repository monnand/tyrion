args=(commandArgs(TRUE))

read.then.normalize = function(filename) {
	data = as.matrix(read.table(filename));
	data = data + abs(min(data));
	data = data / abs(max(data));
	return(data);
}

if (length(args)!=0) {
	for (i in 1:length(args)) {
		eval(parse(text=args[[i]]))
	}
	# d = as.matrix(read.table(input));
	d = read.then.normalize(input);
	postscript(file=output, onefile=FALSE, horizontal=FALSE);
	plot(density(d));
}
