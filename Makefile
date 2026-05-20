BINARY      := httpgo
INSTALL_DIR ?= $(HOME)/.local/bin

.PHONY: build install clean

build:
	go build -o $(BINARY) .

install: build
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

clean:
	@rm -f $(BINARY)
