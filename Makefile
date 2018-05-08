test:
	go build -o testdata/app testdata/app.go
	go test -v
