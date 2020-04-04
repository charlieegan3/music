TAG := $(shell tar -cf - . | md5sum | cut -f 1 -d " ")
PROJECT := music-hugo-rebuilder

build:
	docker build --build-arg="arch=64bit" \
				 -t charlieegan3/$(PROJECT):latest \
				 -t charlieegan3/$(PROJECT):${TAG} .

push: build
	docker push charlieegan3/$(PROJECT):latest
	docker push charlieegan3/$(PROJECT):${TAG}
