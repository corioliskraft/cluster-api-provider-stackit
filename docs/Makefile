MDBOOK ?= mdbook

.PHONY: build
build:
	$(MDBOOK) build .

.PHONY: serve
serve:
	$(MDBOOK) serve . --open

.PHONY: clean
clean:
	rm -rf book
