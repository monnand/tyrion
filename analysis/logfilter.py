#!/usr/bin/python

from __future__ import print_function
import sys

class SegWriter:
    def __init__(self, out):
        self.output = out

    def processOneSeg(self, lines, keys):
        for line in lines:
            self.output.write(line)

class PartialSegDecector:
    def __init__(self, nextProc):
        self.nextProc = nextProc
        self.currentSegNr = 0

    def processOneSeg(self, lines, keys):
        self.currentSegNr += 1
        if len(lines) != len(keys):

            print("segment", self.currentSegNr, "contains partial data\n", file=sys.stderr)
            return
        if self.nextProc is not None:
            self.nextProc.processOneSeg(lines, keys)

def findAllKeys(inputf, partialScan=False):
    ret = set()
    with open(inputf) as f:
        for line in f:
            line = line.strip()
            elems = line.split('\t')
            key = elems[1]
            if key in ret and partialScan:
                return ret
            ret.add(key)
    return ret

def splitIntoSegs(inputf, keys, segProc):
    linebuf = []
    tmpkeys = set()
    with open(inputf) as f:
        for line in f:
            elems = line.split('\t')
            key = elems[1]
            if key in tmpkeys:
                tmpkeys = set()
                segProc.processOneSeg(linebuf, keys)
                linebuf = []
                tmpkeys = set()
            linebuf.append(line)
            tmpkeys.add(key)
        if len(linebuf) > 0:
            segProc.processOneSeg(linebuf, keys)

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print(sys.argv[0], "[input] [output]")
        sys.exit(-1)
    inputf = sys.argv[1]
    o = sys.stdout
    if len(sys.argv) > 2:
        outputf = sys.argv[2]
        o = open(outputf, "w+")
    keys = findAllKeys(inputf)
    outproc = SegWriter(o)
    partial = PartialSegDecector(outproc)
    splitIntoSegs(inputf, keys, partial)
    o.close()

