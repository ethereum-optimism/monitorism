VENV_DIR = .venv
PYTHON = $(VENV_DIR)/bin/python3
PIP = $(VENV_DIR)/bin/pip
REQUIREMENTS = requirements.txt
ENV_DIR = local_data

.PHONY: all venv install clean start

all: venv install

venv:
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

start: venv
	@echo "Starting Jupyter Notebook..."
	$(PYTHON) -m jupyter lab 
