package sqlfs

//go:generate statik -ns sql -m -include=*.sql -f -src ../../sql -dest ../ -p sqlfs
//go:generate gofmt -w .
