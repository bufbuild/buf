ifndef DOCKER_ORG
$(error DOCKER_ORG is not set)
endif
ifndef DOCKER_PROJECT
$(error DOCKER_PROJECT is not set)
endif
ifndef PROJECT
$(error PROJECT is not set)
endif
ifndef GO_MODULE
$(error GO_MODULE is not set)
endif

DOCKER_WORKSPACE_IMAGE := $(DOCKER_ORG)/$(DOCKER_PROJECT)-workspace
DOCKER_WORKSPACE_FILE := Dockerfile.workspace
DOCKER_WORKSPACE_DIR := /workspace

DOCKER_BINS ?=

DOCKERMAKETARGET ?= all

.PHONY: dockercopyworkspacefile
dockercopyworkspacefile:
	cp make/assets/$(DOCKER_WORKSPACE_FILE) $(CURDIR)/$(DOCKER_WORKSPACE_FILE)

pregenerate:: dockercopyworkspacefile

.PHONY: dockerbuildworkspace
dockerbuildworkspace:
	docker build \
		--build-arg PROJECT=$(PROJECT) \
		--build-arg GO_MODULE=$(GO_MODULE) \
		-t $(DOCKER_WORKSPACE_IMAGE) \
		-f $(DOCKER_WORKSPACE_FILE) \
		.

.PHONY: dockermakeworkspace
dockermakeworkspace: dockerbuildworkspace
	docker run -v "$(CURDIR):$(DOCKER_WORKSPACE_DIR)" $(DOCKER_WORKSPACE_IMAGE) make -j 8 $(DOCKERMAKETARGET)

.PHONY: dockerbuild
dockerbuild::

define dockerbinfunc
.PHONY: dockerbuild$(1)
dockerbuild$(1): dockerbuildworkspace
	docker build \
		--build-arg DOCKER_WORKSPACE_IMAGE=$(DOCKER_WORKSPACE_IMAGE) \
		-t $(DOCKER_ORG)/$(1):latest \
		-f Dockerfile.$(1) \
		.

dockerbuild:: dockerbuild$(1)
endef

$(foreach dockerbin,$(DOCKER_BINS),$(eval $(call dockerbinfunc,$(dockerbin))))
