all:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o es-k8s
	docker build -t es-k8s:latest .
