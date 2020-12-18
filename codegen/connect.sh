# Usage: connect.sh <read | write> <clientname> <path-to-spec> <path-to-parent-target-dir>
# Example: connect.sh read userclient ../../user-service2/spec.yaml ../../partner-service/extservices/
rm -rf "$4""$2"
mkdir "$4""$2"

oapi-codegen -package "$2" -generate types "$3" > "$4""$2"/model.gen.go
go run main.go "$1" "$2" "$3" "$4"