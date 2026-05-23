BINARY      := httpgo
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build install clean

help: ## Show this help
	@echo "Available make commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'

build: ## Build the service binary
	go build -o $(BINARY) .

install: build ## Install the binary in $HOME/.local/bin
	@mkdir -p $(INSTALL_DIR)
	@install -m 0755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo ""
	@echo "Installed $(BINARY) -> $(INSTALL_DIR)/$(BINARY)"
	@case ":$$PATH:" in \
	  *":$(INSTALL_DIR):"*) \
	    echo "$(INSTALL_DIR) is already on your PATH. Run '$(BINARY) --help' to get started." ;; \
	  *) \
	    echo ""; \
	    echo "NOTE: $(INSTALL_DIR) is not on your PATH."; \
	    echo "Add it by appending this line to your shell profile (e.g. ~/.zshrc or ~/.bashrc):"; \
	    echo ""; \
	    echo "    export PATH=\"$(INSTALL_DIR):\$$PATH\""; \
	    echo "" ;; \
	esac

clean: ## Delete the installed binaries
	@rm -f $(BINARY)
