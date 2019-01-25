#/bin/sh

# Build time variables for the MySQL connector configuration
DB_NAME=defacto2-inno
DB_USER=root
DB_PASS=
PW_PATH=/path/to/password

go run -ldflags "-X main.dbName=$DB_NAME -X main.dbUser=$DB_USER -X main.dbPass=$DB_PASS -X main.pwPath=$PW_PATH" uuid.go $1