TAG=$(shell git rev-parse HEAD)

image:
	docker build . -t charlieegan3/music:$(TAG) -t charlieegan3/music:latest
	docker push charlieegan3/music:$(TAG)
	docker push charlieegan3/music:latest
