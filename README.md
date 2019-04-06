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

# Build
Run `go build` in zar directory.

# Usage
To create a zar image for a folder, you can run `./zar -w -dir=<folder path>`. e.g. `./zar -w -dir=./test`. If you need to set the output image name and location, add `-o <image path>`. By default it uses `test.img`.

To read a zar image for a folder, you can run `./zar -r`.  If you need to set the input image name and location, add `-img <image path>`. By default it uses `test.img`.

# TODO
1. Add folder support
