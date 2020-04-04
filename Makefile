USER=charlieegan3
PROJECT := $(USER)/music-play-tracker
TAG := $(shell tar -cf - . | md5sum | cut -f 1 -d " ")

login:
	echo "$$DOCKER_PASSWORD" | docker login -u "$(USER)" --password-stdin

build:
	docker build -t $(PROJECT):latest \
				 -t $(PROJECT):$(TAG) .

push: build login
	docker push $(PROJECT):latest
	docker push $(PROJECT):$(TAG)
