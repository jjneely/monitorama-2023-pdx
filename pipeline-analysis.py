#!env python3

import numpy as np
import os

resolution = 300.0 # seconds
fd = open("pipeline.csv")
data = fd.readlines()
raw_data = []
customers = {}

# "@timestamp",cId,StartTs,Duration
# "2023-06-09T21:17:03.431792676Z",1008,1686345422481,684
# "2023-06-09T21:17:03.374365464Z",1008,1686345422481,664

# Read, load, filter initial raw data
for row in data:
    if row.startswith("\"@timestamp"):
        continue

    columns = row.strip().split(",")
    jobDuration = int(columns[3])
    startTs = np.datetime64(int(columns[2]), "ms")
    cId = int(columns[1])
    if jobDuration > 160000000000:
        # Bad data -- junk it
        continue

    if cId not in customers:
        customers[cId] = None
    raw_data.append([cId, startTs, jobDuration])

df = np.array(raw_data)
print("Array shape          : {}".format(df.shape))
_, _, durations = np.split(df, 3, axis=1)
print("Global p50 Duration  : {}".format(np.median(durations)))
print("Global p99 Duration  : {}".format(np.percentile(durations, 99)))
print("Number of CustomerIds: {}".format(len(customers)))

# Calculate per-customer actual p99 duration
c = 0
cActual = []
for cId in customers.keys():
    a = df[np.where(df[:,0]==cId)]
    _, _, durations = np.split(a, 3, axis=1)
    cActual.append([int(cId), np.median(durations), np.percentile(durations, 99)])

cActual = np.array(cActual)
print("Array shape          : {}".format(cActual.shape))
fd = open("pipelines-customer-actual.csv", "w")
fd.write("cId, p50, p99\n")
for r in cActual[cActual[:,2].argsort()]:
    print("Customer %d, p50 %.2f, p99 %.2f" % (r[0], r[1], r[2]))
    fd.write("%d, %f, %f\n" % (r[0], r[1], r[2]))
fd.close()
