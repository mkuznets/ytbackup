#!/usr/bin/env bash

export TZ=Etc/UTC

# parse git revision without .git/object

# .git/HEAD with linke like "ref: refs/heads/master"
head_full=$(cat .git/HEAD)
head_full_elems=(${head_full//:/ })
ref=${head_full_elems[1]}

# get last elements from ref value, i.e. "refs/heads/master"
head_short=(${head_full_elems[1]//// })
head=${head_short[@]: -1:1}

# get hash
full_hash=$(cat .git/$ref)
short_hash=${full_hash:0:7}

# get time stamp
log=$(tail -n1 .git/logs/$ref)
log_elems=(${log// / })

echo "$head"-"$short_hash"
