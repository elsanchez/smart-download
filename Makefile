.PHONY: build install clean test run stop

BINARY_DAEMON=smart-downloadd
BINARY_CLI=smd
INSTALL_DIR=$(HOME)/.local/bin

build:
	@echo "Building binaries..."
	go build -o $(BINARY_DAEMON) ./cmd/smart-downloadd
	go build -o $(BINARY_CLI) ./cmd/smd
	@echo "✓ Binaries built successfully"

install: build
	@echo "Installing binaries to $(INSTALL_DIR)..."
	mkdir -p $(INSTALL_DIR)
	cp $(BINARY_DAEMON) $(INSTALL_DIR)/
	cp $(BINARY_CLI) $(INSTALL_DIR)/
	@echo "✓ Binaries installed"
	@echo ""
	@echo "To start the daemon:"
	@echo "  $(INSTALL_DIR)/$(BINARY_DAEMON)"
	@echo ""
	@echo "Or with systemd:"
	@echo "  make install-systemd"

install-systemd: install
	@echo "Installing systemd service..."
	mkdir -p $(HOME)/.config/systemd/user
	@echo "[Unit]" > $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "Description=Smart Download Daemon" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "After=network.target" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "[Service]" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "Type=simple" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "ExecStart=$(INSTALL_DIR)/$(BINARY_DAEMON)" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "Restart=on-failure" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "RestartSec=5" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "[Install]" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	@echo "WantedBy=default.target" >> $(HOME)/.config/systemd/user/smart-downloadd.service
	systemctl --user daemon-reload
	@echo "✓ Systemd service installed"
	@echo ""
	@echo "To enable and start:"
	@echo "  systemctl --user enable smart-downloadd"
	@echo "  systemctl --user start smart-downloadd"
	@echo ""
	@echo "To check status:"
	@echo "  systemctl --user status smart-downloadd"

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_DAEMON) $(BINARY_CLI)
	@echo "✓ Clean complete"

test:
	@echo "Running tests..."
	go test ./... -v

run:
	@echo "Starting daemon in foreground..."
	./$(BINARY_DAEMON)

stop:
	@echo "Stopping daemon..."
	@pkill -f $(BINARY_DAEMON) || echo "Daemon not running"
