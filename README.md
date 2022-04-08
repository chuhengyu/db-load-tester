# db-load-tester

Cross platform build:
- Linux: GOOS=linux GOARCH=amd64 go build -o db-load-testing-linux
- Mac OS: GOOS=darwin GOARCH=amd64 go build -o db-load-testing-macos
- Windows: GOOS=windows GOARCH=amd64 go build -o db-load-testing-windows

Example usage:
`./db-load-testing-macos -f test.dat -w 20 -n 10000 -p -w 2`
