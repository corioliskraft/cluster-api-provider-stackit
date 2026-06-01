# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ROOT_DIR_RELATIVE := ../..
include $(ROOT_DIR_RELATIVE)/common.mk

# Directories.
BIN_DIR := bin
SHARE_DIR := share

OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
MDBOOK_ARCHIVE_EXT := .tar.gz
MDBOOK_EXTRACT_COMMAND = tar xfz $(MDBOOK_SHARE) -C $(BIN_DIR)
MDBOOK_TARGET := $(ARCH)-unknown-$(OS)-gnu

ifeq ($(OS), darwin)
	MDBOOK_TARGET := $(ARCH)-apple-darwin
endif

ifeq ($(OS), linux)
	MDBOOK_TARGET := $(ARCH)-unknown-linux-gnu
endif

ifeq ($(OS), linux)
ifeq ($(ARCH), arm64)
	MDBOOK_TARGET := $(ARCH)-unknown-linux-musl
endif
endif

ifeq ($(ARCH), amd64)
	MDBOOK_ARCH := x86_64
else ifeq ($(ARCH), arm64)
	MDBOOK_ARCH := aarch64
else
	MDBOOK_ARCH := $(ARCH)
endif

ifeq ($(OS), windows)
	MDBOOK_TARGET := $(MDBOOK_ARCH)-pc-windows-msvc
	MDBOOK_ARCHIVE_EXT := .zip
	MDBOOK_EXTRACT_COMMAND = unzip -o $(MDBOOK_SHARE) -d $(BIN_DIR)
else
	MDBOOK_TARGET := $(MDBOOK_ARCH)$(patsubst $(ARCH)%,%,$(MDBOOK_TARGET))
endif

## --------------------------------------
## Tooling Binaries
## --------------------------------------

$(BIN_DIR):
	mkdir -p $@

$(SHARE_DIR):
	mkdir -p $@

MDBOOK_SHARE := $(SHARE_DIR)/mdbook-$(MDBOOK_VERSION)-$(MDBOOK_TARGET)$(MDBOOK_ARCHIVE_EXT)
$(MDBOOK_SHARE): ../../versions.mk $(SHARE_DIR)
	curl -fsSL -o $@ "https://github.com/rust-lang/mdBook/releases/download/$(MDBOOK_VERSION)/mdbook-$(MDBOOK_VERSION)-$(MDBOOK_TARGET)$(MDBOOK_ARCHIVE_EXT)"

MDBOOK := $(BIN_DIR)/mdbook
$(MDBOOK): $(BIN_DIR) $(MDBOOK_SHARE)
	$(MDBOOK_EXTRACT_COMMAND)
	chmod +x $@
	touch -m $@

MDBOOK_EMBED := $(BIN_DIR)/mdbook-embed
$(MDBOOK_EMBED): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ sigs.k8s.io/cluster-api/hack/tools/mdbook/embed

MDBOOK_RELEASELINK := $(BIN_DIR)/mdbook-releaselink
$(MDBOOK_RELEASELINK): $(BIN_DIR) go.mod go.sum
	go build -tags=tools -o $@ sigs.k8s.io/cluster-api/hack/tools/mdbook/releaselink

.PHONY: clean
clean: ## Remove all tools
	rm -rf bin
