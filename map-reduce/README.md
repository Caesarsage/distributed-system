# Map reduce

Map reduce is used for computing large data in parrallel where it is broken into two phase, map and reduce phase. The data is first broken into chunks in the map phase and combine together in the reduce phase to give a final result.


## counting word example

1. Master creates map tasks (one per input file)
2. Workers request tasks → get map tasks
3. Workers process data with map function
4. Workers report map results back
5. Once ALL maps are done, reduce phase begins
6. Workers request tasks → get reduce tasks
7. Workers process with reduce function
8. Workers report reduce results
9. Job complete!
