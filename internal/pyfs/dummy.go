package pyfs

//go:generate statik -ns python -m -include=*.py -f -src ../../python -dest ../ -p pyfs
//go:generate gofmt -w .
