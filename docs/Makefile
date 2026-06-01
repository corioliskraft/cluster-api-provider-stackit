ROOT_DIR_RELATIVE := ..
include $(ROOT_DIR_RELATIVE)/common.mk

# Directories.
MDBOOK := $(TOOLS_BIN_DIR)/mdbook
MDBOOK_EMBED := $(TOOLS_BIN_DIR)/mdbook-embed
MDBOOK_RELEASELINK := $(TOOLS_BIN_DIR)/mdbook-releaselink

BOOK_DEPS := $(BOOK_SRCS) $(MDBOOK) $(MDBOOK_EMBED) $(MDBOOK_RELEASELINK)
export PATH := $(abspath $(TOOLS_BIN_DIR)):$(PATH)

.PHONY: build
build: $(BOOK_DEPS) ## Build the book
	$(MDBOOK) build .

.PHONY: serve
serve:
	$(MDBOOK) serve . --open

.PHONY: clean
clean:
	rm -rf book
