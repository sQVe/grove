test:
	go test -json -short -tags=!integration ./... 2>&1 | tdd-guard-go -project-root $(PWD)
