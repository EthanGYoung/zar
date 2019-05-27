# Introduction
Zar is a software for collecting many files into one archive img. It is optimized for golang and read-only mmap random read. Since this is used for read-only image file system, it doesn't contain various file system parameters such as time stamps, ownership and permissions.

# Structure
zar img file looks like this:
```
| file 1 data |...| file n data | files metadata lists | files metadata start offset
```
files metadata is a list consisting of the following structs:
```go
type fileMetadata struct {
	Begin int64
	End int64
	Name string
}
```
Begin indicated the file start offset.
End indicated the file end offset.
Name indicated the file name. (Note: In the next version we will support folders. If there is a file bar.txt in folder foo, the Name field will be "foo/bar.txt")

# Enviornment Setup
1) Add the zar directory to the GOPATH by running (if added at home dir):
```
export GOPATH=$HOME/zar
```

2) Add the bin directory to golang enviorment by running (if added at home dir):
```
export GOBIN=$HOME/zar/bin
```

# Build
Run `go build src/zar.main.go`.

# Usage
To create a zar image for a folder, you can run `./bin/main -w -dir=<folder path>`. e.g. `./bin/main -w -dir=./test`. If you need to set the output image name and location, add `-o <image path>`. By default it uses `test.img`.

To read a zar image for a folder, you can run `./bin/main -r`.  If you need to set the input image name and location, add `-img <image path>`. By default it uses `test.img`.
