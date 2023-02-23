build:
	go build -ldflags "-s -w -X github.com/chriskim06/kubectl-topui/internal/cmd.tag=DEV" -o out/kubectl-topui

clean:
	rm -r out
