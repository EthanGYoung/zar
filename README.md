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
    Link string
    ModTime int64
    Type fileType
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
Run `go build src/zar/main.go`.

# Usage
To create a zar image for a folder, you can run `./bin/main -w -dir=<folder path> -pagealign`. e.g. `./bin/main -w -dir=./test -pagealign`. If you need to set the output image name and location, add `-o <image path>`. By default it uses `test.img`.

To read a zar image for a folder, you can run `./bin/main -r`.  If you need to set the input image name and location, add `-img <image path>`. By default it uses `test.img`.

# Flags
* `-w`: write mode
* flags only for write mode
    * `-dir=<dir>`: the root dir to be archived
    * `-o=<file_name>`: output image file name, by deafult it is "test.img"
    * `-pagealign`: IMPORTANT flag. It is necessary for imgfs mmap feature. Please enable it every time when you create an imgfs image. All start offset will be aligned to 4K location.
* `-r`: read mode
* flags only for read mode
    * `-detail`: Output all file content when reading from the image.

* other flags
    * `-config`, `-configPath`, `-configFormat`.

# ContainerFS Image Generator

## Python Docker SDK
To install Python Docker SDK, please run `pip install docker`, this is required by CFS Image Generator.

## Use
Usually when we build a docker image, we firstly create a folder with Dockerfile and some other files that may be used when building the image. Let's assume the folder name is `open-lambda-image`.

In this folder you can run `docker build -t open-lambda .` to create a docker image based on the rules defined in the Dockerfile. The created image name is `open-lambda`.

ContainerFS image generator is used for converting traditional docker image to containerFS docker image which is designed for gVisor containerFS. To use the tool, you should run the following command:
```
# sudo python cfs_generator.py <docker image name> <destination folder> e.g.
sudo python cfs_generator.py open-lambda /tmp/open-lambda-cfs
```
**Note: before running the command, please make sure you have used `go build` to build tar tool in the current folder. binary file `main` should show up in the current folder.**

CFS image generator will create necessary files for building CFS images in the destination folder. Then you can enter the destination folder (in the previous example that is `/tmp/open-lambda-cfs`) and build it. e.g. `docker build -t open-lambda-cfs /tmp/open-lambda-cfs`.

Now you can use open-lambda-cfs image as a container fs image for booting gvisor.
