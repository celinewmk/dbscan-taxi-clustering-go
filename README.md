# Taxi Clustering using Go

## Instructions
1. Start terminal and cd into the directory
2. run `go run map.go`.

The values of MinPts, eps, N, and consumer threads can also be changed.

## Description
In the Go implementation of the Taxi Clustering, the DBSCAN algorithm runs concurrently on partitions of the dataset. To get partitions, the geographical area is divided into a grid of NxN. For instance, the area below is a 4x4 grid that would have to run 16 concurrent DBSCAN threads.

<img style="margin-left: auto; margin-right:auto" width="297" alt="Screenshot 2022-03-18 at 13 03 32" src="https://user-images.githubusercontent.com/71091659/159049316-a835a86b-81fa-4752-b9a7-339d2b16bfc3.png">


Link to an explanation of the DBSCAN algorithm: https://en.wikipedia.org/wiki/DBSCAN

The dataset I am using is a CSV file containing all the trip records for January 15, 2009 between 9am and 9pm in New York City.

## Algorithm based on MapReduce pattern
The parallel DBSCAN algorithm follows those steps:
1. Map the data into overlapping partitions (the overlap with other partitions must be at least equals to eps along its border)
2. Apply the DBSCAN algorithm over each partition.
3. Reduce the results by collecting the clusters from all partitions. Intersecting clusters must be merged. (This step is not implemented here)

## Producer-consumer pattern
The concurrent version of the DBSCAN algorithm is based on the producer-consumer pattern.

The producer will be in the main thread and will simply send jobs to a channel; each job being a clustering to be done on one partition of the data. The job instance will include a slice of GPS
coordinates and the value of the parameters required to execute the DBSCAN algorithm on this set of
points (minPts, eps, offset).

All the jobs are processed by a certain number of consumers, each running in its own thread. When a
consumer is done with one job, it then consumes the next job. When there is no more job to consume,
then all consumer threads terminate.

## Analysis (with MinPts = 5 and eps = 0.0003)
- N = 2 and 4 consumer threads → Execution time : 22.831198978s
- N = 4 and 4 consumer threads → Execution time : 7.653588536s
- N = 4 and 10 consumer threads → Execution time : 7.961181397s
- N = 10 and 4 consumer threads → Execution time : 1.890908587s
- N = 10 and 10 consumer threads → Execution time: 1.95273925s
- N = 10 and 50 consumer threads → Execution time : 1.966321561s
- N = 20 and 10 consumer threads → Execution time : 1.520464409s
- N = 20 and 50 consumer threads → Execution time : 1.538839004s
- N = 20 and 200 consumer threads → Execution time: 1.559544214s
- N = 20 and 2 consumer threads → Execution time: 1.886381764s
- N = 25 and 10 consumer threads → Execution time: 1.67255536s
- N = 25 and 20 consumer threads → Execution time: 1.712295995s
- N = 50 and 20 consumer threads → Execution time: 3.666361951s
- N = 75 and 20 consumer threads → Execution time : 7.737341648s

After running our program on different N and consumer threads, we can observe that the program is much faster when the N is not too small or not too large. For example, for N = 2 or N = 4, the execution time is higher than 7s, and when N = 50 or N = 75, the execution time is higher than 3s. The only experiments where we were able to obtain execution times of about 1-2s were when N = 10, 20 or 25. We can conlude that when N is too large, there are partitions that end up with 0 points and are then processed unnecessarily, and when N is too small, partitions end up with a high number of points that take time to process.
Similarly, for the number of consumer threads, we should not take a number that is too small or too large. We can observe that when the number of consumer threads is very high, the execution time is slightly higher; and when the number of consumer threads is too low, the program is slower (example: experimental results with N = 20). 
According to our experiments, to have the optimal configuration, we should take an N and a number of consumer threads that are similar to N = 20 and 10 consumer threads.

## Example results (MinPts = 5, eps = 0.0003, N = 4 and 10 consumer threads)
```
Number of points: 232050
SW:(40.700000 , -74.000000)
NE:(40.800000 , -73.930000) 

Partition with N = 4 and 10 consumer threads

Partition   30000000 : [   0,    17]
Partition   20000000 : [   9,   272]
Partition   10000000 : [  12,   540]
Partition   21000000 : [   6,   288]
Partition   31000000 : [   7,   226]
Partition   32000000 : [  15,  1192]
Partition    3000000 : [   4,  2015]
Partition   33000000 : [  19,  2187]
Partition          0 : [  33,  6514]
Partition   11000000 : [  20, 14022]
Partition   13000000 : [  14, 15370]
Partition   23000000 : [  20, 16535]
Partition   22000000 : [  15, 19940]
Partition    2000000 : [  25, 31599]
Partition    1000000 : [  18, 38774]
Partition   12000000 : [   9, 56611]

Execution time: 7.961181397s of 206102 points
Number of CPUs: 8
```


