#/bin/sh
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
DB_PASS=password
#
# Build time variables to local directories
#
# Path to file downloads named as UUID
PATH_UUID=/opt/webapp/uuid/
# Path to image previews and thumbnails
PATH_IMAGE=/opt/webapp/images/
# Path to webapp generated files such as JSON/XML
PATH_FILES=/opt/webapp/files/

go build -ldflags "\
-X main.dbName=$DB_NAME \
-X main.dbUser=$DB_USER \
-X main.pwPath=$PW_PATH \
-X main.dbPass=$DB_PASS \
-X main.pathUUID=$PATH_UUID \
-X main.pathImageBase=$PATH_IMAGE \
-X main.pathFilesBase=$PATH_FILES \
" $@ uuid.go