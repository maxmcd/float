



build: *.go Dockerfile
	docker build -t float .

push: build
	docker tag float maxmcd/float
	docker push maxmcd/float
