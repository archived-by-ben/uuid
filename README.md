# uuid
_Defacto2 manager of UUID named files_

[![Go Report Card](https://goreportcard.com/badge/github.com/Defacto2/uuid)](https://goreportcard.com/report/github.com/Defacto2/uuid) 
[![Build Status](https://travis-ci.org/Defacto2/uuid.svg?branch=master)](https://travis-ci.org/Defacto2/uuid)

[Created in Go](https://golang.org/doc/install), to build from source.

Clone this repository.

```sh
git clone https://github.com/Defacto2/uuid.git
```

Edit the `build.sh` shell script to supply build time variables, save then run.

```sh
nano build.sh
```

```sh
#!/bin/sh
#
# Build time variables for the database connector configuration
#
# Database name
DB_NAME=defacto2-inno
# Database login username
DB_USER=root
# Path to a text file containing the database login user password
PW_PATH=/path/to/password
# Database login fallback password when text file is not found [should normally be left blank]
DB_PASS=
#
# Build time variables to local directories
#
# Path to file downloads named as UUID
PATH_UUID=/opt/webapp/uuid/
# Path to image previews and thumbnails
PATH_IMAGE=/opt/webapp/images/
# Path to webapp generated files such as JSON/XML
PATH_FILES=/opt/webapp/files/

```

Now you can run and tryout the package.
```sh
./build.sh run --version; ./build.sh run -h
```

Install the package.
```sh
./build.sh install
uuid --version
uuid -h
```

Build the package (may require dependencies).
```sh
./build.sh
./uuid --version
./uuid -h
```

Install dependencies.

```sh
go get github.com/docopt/docopt-go
go get github.com/dustin/go-humanize
go get github.com/go-sql-driver/mysql
```