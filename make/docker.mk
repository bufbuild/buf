ifndef DOCKER_IMAGE
$(error DOCKER_IMAGE is not set)
endif
ifndef DOCKER_FILE
$(error DOCKER_FILE is not set)
endif
ifndef DOCKER_DIR
$(error DOCKER_DIR is not set)
endif

.PHONY: dockerbuild
dockerbuild:
	docker build -t $(DOCKER_IMAGE) -f $(DOCKER_FILE) .

.PHONY: dockermake
dockermake: dockerbuild
	docker run -v "$(CURDIR):$(DOCKER_DIR)" $(DOCKER_IMAGE) make -j 8
