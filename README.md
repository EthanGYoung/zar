# Introduction
Zar is a software for collecting many files into one archive img. It is optimized for golang and read-only mmap random read. Since this is used for read-only image file system, it doesn't contain various file system parameters such as time stamps, ownership and permissions.

# Structure
zar img file looks like this:
```
| file 1 data |...| file n data | files metadata | files metadata start offset
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

# TODO
1. Add folder support

