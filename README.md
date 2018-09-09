# go-hashicorp-raft-http
A reference implementation by using the Hashicorp Raft lib.

# build
```
go build -o bin/go-hashicorp-raft-http cmd/*
```

# up and running
```
bin/go-hashicorp-raft-http -id matching1 -raftDir ./snapshots/matching1 -raftAddr :7777  -httpAddr :8887
bin/go-hashicorp-raft-http -id matching2 -raftDir ./snapshots/matching2 -raftAddr :7778  -httpAddr :8888 -joinHttpAddr :8887
bin/go-hashicorp-raft-http -id matching3 -raftDir ./snapshots/matching3 -raftAddr :7779  -httpAddr :8889 -joinHttpAddr :8887
```

# testing

## set
```
curl -XPOST localhost:8887/key -d '{"x": "3"}'
curl -XPOST localhost:8887/key -d '{"y": "1"}'
curl -XPOST localhost:8887/key -d '{"y": "9"}'
curl -XPOST localhost:8887/key -d '{"x": "2"}'
curl -XPOST localhost:8887/key -d '{"x": "0"}'
curl -XPOST localhost:8887/key -d '{"y": "7"}'
curl -XPOST localhost:8887/key -d '{"x": "5"}'
curl -XPOST localhost:8887/key -d '{"x": "4"}'
curl -XPOST localhost:8887/key -d '{"z": "1"}'
curl -XPOST localhost:8887/key -d '{"z": "2"}'

curl -XPOST localhost:8887/key -d '{"z": "3"}'
curl -XPOST localhost:8887/key -d '{"z": "4"}'
curl -XPOST localhost:8887/key -d '{"z": "5"}'
curl -XPOST localhost:8887/key -d '{"z": "6"}'
curl -XPOST localhost:8887/key -d '{"z": "7"}'
curl -XPOST localhost:8887/key -d '{"z": "8"}'
curl -XPOST localhost:8887/key -d '{"z": "9"}'
curl -XPOST localhost:8887/key -d '{"z": "10"}'
curl -XPOST localhost:8887/key -d '{"z": "11"}'
curl -XPOST localhost:8887/key -d '{"z": "12"}'

curl -XPOST localhost:8887/key -d '{"z": "13"}'
curl -XPOST localhost:8887/key -d '{"z": "14"}'
curl -XPOST localhost:8887/key -d '{"z": "15"}'
```

## get
```
curl -XGET localhost:8887/key/x
```

## leave
```
curl -XPOST localhost:8887/leave -d '{"id": "matching2","addr":":7778"}'
```

## raft State
```
curl -XGET localhost:8887/raftstate
```

## snapshot
```
curl -XGET localhost:8887/snapshot
```

## test snapshot when kill process
```
ps aux | grep go-hashicorp-raft-http
kill 58443
```

# Todo
* etcd
* zookeeper

# Reference
https://github.com/otoolep/hraftd
