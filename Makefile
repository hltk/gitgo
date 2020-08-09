all: gitgo

gitgo: go.mod *.go
	go build -o gitgo

clean:
	rm -f gitgo
