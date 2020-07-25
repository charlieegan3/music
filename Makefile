TAG=$(shell git rev-parse HEAD)

ifndef DOCKER_USERNAME
$(error DOCKER_USERNAME is not set)
endif
ifndef DOCKER_PASSWORD
$(error DOCKER_PASSWORD is not set)
endif

docker-login:
	docker login -u $(DOCKER_USERNAME) -p $(DOCKER_PASSWORD)

image: docker-login
	docker build . -t charlieegan3/music:$(TAG) -t charlieegan3/music:latest
	docker push charlieegan3/music:$(TAG)
	docker push charlieegan3/music:latest
