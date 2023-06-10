#!env python3

import numpy as np

resolution = 300.0 # seconds
fd = open("pipeline.csv")
data = fd.readlines()
raw_data = []

# "@timestamp",cId,StartTs,Duration
# "2023-06-09T21:17:03.431792676Z",1008,1686345422481,684
# "2023-06-09T21:17:03.374365464Z",1008,1686345422481,664

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

    raw_data.append([cId, startTs, jobDuration])

df = np.array(raw_data)
print("Array shape        : {}".format(df.shape))
_, _, durations = np.split(df, 3, axis=1)
print("Global p50 Duration: {}".format(np.median(durations)))
print("Global p99 Duration: {}".format(np.percentile(durations, 99)))

