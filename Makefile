run: build
	docker run -it --rm -p 8080:8080 stojg/track-lister

build:
	docker build . -t stojg/track-lister

push: build
	docker build . -t stojg/track-lister:latest -t stojg/track-lister:$(shell git rev-parse --verify HEAD)
	docker push stojg/track-lister:latest
	docker push stojg/track-lister:$(shell git rev-parse --verify HEAD)