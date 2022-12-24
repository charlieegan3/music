FILE_PATTERN := 'html\|go\|sql\|Makefile'

dev-server:
	find . | grep $(FILE_PATTERN) | entr -r bash -c 'clear; pkill -f cmd/utils/tool.go; go run cmd/utils/tool.go'

binary:
	go build
