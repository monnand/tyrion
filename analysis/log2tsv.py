#!/usr/bin/python
import sys
class LogStore:
    def __init__(self):
        self.store = {}
    def append(self, key, value):
        if not self.store.has_key(key):
            self.store[key] = []
        self.store[key].append(value)

    def dump(self, out):
        N = 0
        for k, v in self.store.items():
            if N <= 0:
                N = len(v)
            elif N > 0 and len(v) != N:
                sys.stderr.write(" ".join([k, "has", str(len(v)), "values. but should have", str(N), "values"]))
                return
        keys = self.store.keys()
        for i in xrange(0, N):
            v = []
            for key in keys:
                v.append(self.store[key][i])
            out.write("\t".join(v))
            out.write("\n")

class LogReader:
    def __init__(self, inputf):
        self.store = LogStore()
        self.inputf = inputf

    def load(self):
        with open(self.inputf) as f:
            for line in f:
                elems = line.split('\t')
                key = elems[1]
                value = elems[2]
                self.store.append(key,value)

    def dump(self, out):
        self.store.dump(out)

if __name__ == "__main__":
        inputf = sys.argv[1]
        lr = LogReader(inputf)
        lr.load()
        lr.dump(sys.stdout)
