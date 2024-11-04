VENV_DIR = .venv
PYTHON = $(VENV_DIR)/bin/python3
PIP = $(VENV_DIR)/bin/pip
REQUIREMENTS = requirements.txt
ENV_DIR = local_data

.PHONY: all venv install clean start

all: venv install

venv:
	@which python3 > /dev/null || { echo "Error: python3 not found! Please install Python 3."; exit 1; }
	@if [ ! -d "$(VENV_DIR)" ]; then \
		echo "Creating virtual environment..."; \
		python3 -m venv $(VENV_DIR); \
		echo "To activate the virtual environment, run:"; \
		echo "source $(VENV_DIR)/bin/activate"; \
	else \
		echo "Virtual environment already exists."; \
		echo "To activate the virtual environment, run:"; \
		echo "source $(VENV_DIR)/bin/activate"; \
	fi

install: venv
	@echo "Installing requirements..."
	$(PIP) install -r $(REQUIREMENTS)

clean:
	@echo "Cleaning up..."
	rm -rf $(VENV_DIR)

start: install
	@echo "Starting Jupyter Notebook..."
	$(PYTHON) -m jupyter lab 
