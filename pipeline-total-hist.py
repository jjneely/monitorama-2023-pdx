#!env python3

resolution = 300.0 # seconds
fd = open("pipeline.dat")
data = fd.readlines()
histogram = {} # le value => count

for row in data:
    if row.startswith("\"@"):
        continue

    sample = float(row.strip().split(",")[3])
    #if sample > 168625862064:
        # Timestamp?
    if sample > 86400:
        continue
    bucket = sample // resolution

    key = (bucket * resolution)
    if key in histogram:
        histogram[key] = histogram[key] + 1
    else:
        histogram[key] = 1

for key in sorted(histogram.keys()):
    print("%f\t%f" % (key, histogram[key]/len(data)))
