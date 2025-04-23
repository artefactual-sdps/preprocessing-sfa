PROJECT := preprocessing-sfa
MAKEDIR := hack/make
SHELL   := /bin/bash

.DEFAULT_GOAL := help
.PHONY: *

DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif

include hack/make/bootstrap.mk
include hack/make/dep_ent.mk
include hack/make/dep_go_enum.mk
include hack/make/dep_golangci_lint.mk
include hack/make/dep_gomajor.mk
include hack/make/dep_gosec.mk
include hack/make/dep_gotestsum.mk
include hack/make/dep_mockgen.mk
include hack/make/dep_shfmt.mk
include hack/make/dep_tparse.mk
include hack/make/enums.mk

# Lazy-evaluated list of tools.
TOOLS = $(ENT) \
	$(GOLANGCI_LINT) \
	$(GOMAJOR) \
	$(GOSEC) \
	$(GOTESTSUM) \
	$(MOCKGEN) \
	$(SHFMT) \
	$(TPARSE)

define NEWLINE


endef

IGNORED_PACKAGES := \
	github.com/artefactual-sdps/preprocessing-sfa/hack/% \
	github.com/artefactual-sdps/preprocessing-sfa/internal/%/fake \
	github.com/artefactual-sdps/preprocessing-sfa/internal/enums \
	github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/db \
	github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/db/% \
	github.com/artefactual-sdps/preprocessing-sfa/internal/persistence/ent/schema

PACKAGES := $(shell go list ./...)
TEST_PACKAGES := $(filter-out $(IGNORED_PACKAGES),$(PACKAGES))
TEST_IGNORED_PACKAGES := $(filter $(IGNORED_PACKAGES),$(PACKAGES))

export PATH:=$(GOBIN):$(PATH)

deps: # @HELP List available module dependency updates.
deps: $(GOMAJOR)
	gomajor list

env: # @HELP Print Go env variables.
env:
	go env

fmt: # @HELP Format the project Go files with golangci-lint.
fmt: FMT_FLAGS ?=
fmt: $(GOLANGCI_LINT)
	golangci-lint fmt $(FMT_FLAGS)

gen-ent: # @HELP Generate Ent assets.
gen-ent: $(ENT)
	ent generate ./internal/persistence/ent/schema \
		--feature sql/versioned-migration \
		--target=./internal/persistence/ent/db

gen-mock: # @HELP Generate mocks.
gen-mock: $(MOCKGEN)
	mockgen -typed -destination=./internal/amss/fake/mock_client.go -package=fake github.com/artefactual-sdps/preprocessing-sfa/internal/amss Client
	mockgen -typed -destination=./internal/fformat/fake/mock_identifier.go -package=fake github.com/artefactual-sdps/preprocessing-sfa/internal/fformat Identifier
	mockgen -typed -destination=./internal/fvalidate/fake/mock_validator.go -package=fake github.com/artefactual-sdps/preprocessing-sfa/internal/fvalidate Validator
	mockgen -typed -destination=./internal/persistence/fake/mock_service.go -package=fake github.com/artefactual-sdps/preprocessing-sfa/internal/persistence Service

gosec: # @HELP Run gosec security scanner.
gosec: GOSEC_VERBOSITY ?= "-terse"
gosec: $(GOSEC)
	gosec \
		$(GOSEC_VERBOSITY) \
		-exclude-dir=hack \
		./...

help: # @HELP Print this message.
	echo "TARGETS:"
	grep -hE '^.*:.*?# *@HELP' $(MAKEFILE_LIST) | sort | \
	    awk 'BEGIN {FS = ":.*?# *@HELP"}; { printf "  %-30s %s\n", $$1, $$2 };'

lint: # @HELP Lint the project Go files with golangci-lint.
lint: LINT_FLAGS ?= --timeout=5m --fix --output.text.colors
lint: $(GOLANGCI_LINT)
	golangci-lint run $(LINT_FLAGS)

list-ignored-packages: # @HELP Print a list of packages ignored in testing.
list-ignored-packages:
	$(foreach PACKAGE,$(TEST_IGNORED_PACKAGES),@echo $(PACKAGE)$(NEWLINE))

list-tested-packages: # @HELP Print a list of packages being tested.
list-tested-packages:
	$(foreach PACKAGE,$(TEST_PACKAGES),@echo $(PACKAGE)$(NEWLINE))

pre-commit: # @HELP Check that code is ready to commit.
pre-commit:
	ENDURO_PP_INTEGRATION_TEST=1 $(MAKE) -j \
	fmt \
	gen-enums \
	gosec GOSEC_VERBOSITY="-quiet" \
	lint \
	shfmt \
	test-race

shfmt: SHELL_PROGRAMS := $(shell find $(CURDIR)/hack -name *.sh)
shfmt: $(SHFMT) # @HELP Run shfmt to format shell programs in the hack directory.
	shfmt \
		--list \
		--write \
		--diff \
		--simplify \
		--language-dialect=posix \
		--indent=0 \
		--case-indent \
		--space-redirects \
		--func-next-line \
			$(SHELL_PROGRAMS)

test: # @HELP Run all tests and output a summary using gotestsum.
test: TFORMAT ?= short
test: GOTEST_FLAGS ?=
test: COMBINED_FLAGS ?= $(GOTEST_FLAGS) $(TEST_PACKAGES)
test: $(GOTESTSUM)
	gotestsum --format=$(TFORMAT) -- $(COMBINED_FLAGS)

test-ci: # @HELP Run all tests in CI with coverage and the race detector.
test-ci:
	ENDURO_PP_INTEGRATION_TEST=1 $(MAKE) test GOTEST_FLAGS="-race -coverprofile=covreport -covermode=atomic"

test-race: # @HELP Run all tests with the race detector.
test-race:
	$(MAKE) test GOTEST_FLAGS="-race"

test-tparse: # @HELP Run all tests and output a coverage report using tparse.
test-tparse: $(TPARSE)
	go test -count=1 -json -cover $(TEST_PACKAGES) | tparse -follow -all -notests

tools: # @HELP Install tools.
tools: $(TOOLS)
