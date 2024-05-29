ENUMS := \
 	internal/enums/sip_type_enum.go

$(ENUMS): GO_ENUM_FLAGS=--marshal --names --ptr --flag --sql --template=$(CURDIR)/hack/make/enums.tmpl

gen-enums: $(ENUMS) # @HELP Generate go-enum assets.

%_enum.go: %.go $(GO_ENUM) hack/make/enums.mk hack/make/enums.tmpl
	go-enum -f $*.go $(GO_ENUM_FLAGS)
