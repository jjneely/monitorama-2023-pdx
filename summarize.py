#!env python3

resolution = 0.1 # seconds
fd = open("graphite.dat")
data = fd.readlines()
histogram = {} # le value => count

for row in data:
    sample = float(row.strip())
    bucket = sample // resolution

    key = (bucket * resolution)
    if key in histogram:
        histogram[key] = histogram[key] + 1
    else:
        histogram[key] = 1

for key in sorted(histogram.keys()):
    print("%f\t%f" % (key, histogram[key]/10000))
